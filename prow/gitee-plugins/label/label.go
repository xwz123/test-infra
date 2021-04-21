package label

import (
	"fmt"
	"regexp"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

var (
	defaultLabels    = []string{"kind", "priority", "sig"}
	labelRegex       = regexp.MustCompile(`(?m)^/(kind|priority|sig)\s*(.*?)\s*$`)
	removeLabelRegex = regexp.MustCompile(`(?m)^/remove-(kind|priority|sig)\s*(.*?)\s*$`)
)

type giteeClient interface {
	GetRepoLabels(owner, repo string) ([]sdk.Label, error)
	GetIssueLabels(org, repo, number string) ([]sdk.Label, error)
	GetPRLabels(org, repo string, number int) ([]sdk.Label, error)

	AddIssueLabel(org, repo, number, label string) error
	RemoveIssueLabel(org, repo, number, label string) error

	AddMultiIssueLabel(org, repo, number string, label []string) error
	AddMultiPRLabel(org, repo string, number int, label []string) error
	RemovePRLabel(org, repo string, number int, label string) error

	CreatePRComment(org, repo string, number int, comment string) error
	CreateIssueComment(org, repo string, number string, comment string) error
}

type label struct {
	ghc             giteeClient
	getPluginConfig plugins.GetPluginConfig
}

//NewLabel create a label plugin
func NewLabel(f plugins.GetPluginConfig, gec giteeClient) plugins.Plugin {
	return &label{ghc: gec, getPluginConfig: f}
}

func (l *label) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The label plugin provides commands that add or remove certain types of labels. Labels of the following types can be manipulated: 'kind/*', 'priority/*', 'sig/*'.",
		Config: map[string]string{
			"": configString(defaultLabels),
		},
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/[remove-](kind|priority|sig|label) <target>",
		Description: "Applies or removes a label from one of the recognized types of labels.",
		Featured:    false,
		WhoCanUse:   "Anyone can trigger this command on a PR.",
		Examples:    []string{"/kind bug", "/sig testing", "/priority high"},
	})
	return pluginHelp, nil
}

func (l *label) PluginName() string {
	return "label"
}

func (l *label) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (l *label) RegisterEventHandler(p plugins.Plugins) {
	p.RegisterNoteEventHandler(l.PluginName(), l.handleNoteEvent)
	p.RegisterPullRequestHandler(l.PluginName(), l.handlePullRequestEvent)
}

func (l *label) getLabelCfg() (*configuration, error) {
	cfg := l.getPluginConfig(l.PluginName())
	if cfg == nil {
		return nil, fmt.Errorf("can't find the configuration")
	}
	lCfg, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}
	return lCfg, nil
}

func (l *label) orgRepoCfg(org, repo string) (*labelCfg, error) {
	cfg, err := l.getLabelCfg()
	if err != nil {
		return nil, err
	}
	labelCfg := cfg.labelFor(org, repo)
	if labelCfg == nil {
		return nil, fmt.Errorf("no label plugin config for this repo:%s/%s", org, repo)
	}
	return labelCfg, nil
}

func (l *label) handlePullRequestEvent(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	tp := plugins.ConvertPullRequestAction(e)
	if tp == github.PullRequestActionSynchronize {
		return l.handleClearLabel(e, log)
	}
	return nil
}

func (l *label) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	ne := gitee.NewNoteEventWrapper(e)
	if !ne.IsCreatingCommentEvent() {
		log.Debug("Event is not a creation of a comment, skipping.")
		return nil
	}

	body := ne.GetComment()
	labelMatches := labelRegex.FindAllStringSubmatch(body, -1)
	removeLabelMatches := removeLabelRegex.FindAllStringSubmatch(body, -1)
	if len(labelMatches) == 0 && len(removeLabelMatches) == 0 {
		return nil
	}

	needAddLabels := getLabelsFromREMatches(labelMatches)
	needRemoveLabels := getLabelsFromREMatches(removeLabelMatches)
	org, repo := ne.GetOrgRep()
	var neh handler
	nh := noteHandle{
		client:       l.ghc,
		log:          log,
		addLabels:    needAddLabels,
		removeLabels: needRemoveLabels,
		org:          org,
		repo:         repo,
	}
	if ne.IsPullRequest() {
		number := gitee.NewPRNoteEvent(e).GetPRNumber()
		neh = &prNoteHandle{noteHandle: nh, number: number}
	} else if ne.IsIssue() {
		number := gitee.NewIssueNoteEvent(e).GetIssueNumber()
		neh = &issueNoteHandle{noteHandle: nh, number: number}
	} else {
		return nil
	}

	return neh.handleComment()
}

// Get Labels from Regexp matches
func getLabelsFromREMatches(matches [][]string) (labels []string) {
	for _, match := range matches {
		for _, label := range strings.Split(match[0], " ")[1:] {
			label = strings.ToLower(match[1] + "/" + strings.TrimSpace(label))
			labels = append(labels, label)
		}
	}
	return
}

func configString(labels []string) string {
	var formattedLabels []string
	for _, label := range labels {
		formattedLabels = append(formattedLabels, fmt.Sprintf(`"%s/*"`, label))
	}
	if len(formattedLabels) > 0 {
		return fmt.Sprintf(
			"The label plugin will work on %s and %s labels.",
			strings.Join(formattedLabels[:len(formattedLabels)-1], ", "), formattedLabels[len(formattedLabels)-1],
		)
	}
	return ""
}
