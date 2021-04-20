package label

import (
	"fmt"
	"regexp"
	"strings"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
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

func NewLabel(f plugins.GetPluginConfig, gec giteeClient) plugins.Plugin {
	return &label{ghc: gec, getPluginConfig: f}
}

func (l *label) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	var labels []string
	labels = append(labels, defaultLabels...)
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The label plugin provides commands that add or remove certain types of labels. Labels of the following types can be manipulated: 'kind/*', 'priority/*', 'sig/*'.",
		Config: map[string]string{
			"": configString(labels),
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
	labelCfg := cfg.LabelFor(org, repo)
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

	if ne.IsPullRequest() {
		return l.handlePRCommentEvent(gitee.NewPRNoteEvent(e), labelMatches, removeLabelMatches, log)
	}

	if ne.IsIssue() {
		return l.handleIssueCommentEvent(gitee.NewIssueNoteEvent(e), labelMatches, removeLabelMatches, log)
	}

	return nil
}

func (l *label) handlePRCommentEvent(e gitee.PRNoteEvent, addMatches, rmMatches [][]string, log *logrus.Entry) error {
	org, repo := gitee.GetOwnerAndRepoByEvent(e.NoteEvent)
	number := e.GetPRNumber()
	action := &prNoteAction{l.ghc, org, repo, number}
	repoLabels, prLabels, err := l.getRepoAndCommentOBJLabels(action, org, repo)
	if err != nil {
		return err
	}
	if len(rmMatches) > 0 {
		removeMatchLabels(rmMatches, prLabels, action, log)
	}
	if len(addMatches) > 0 {
		return addMatchLabels(addMatches, prLabels, repoLabels, action, log)
	}
	return nil
}

func (l *label) handleIssueCommentEvent(e gitee.IssueNoteEvent, addMatches, rmMatches [][]string, log *logrus.Entry) error {
	org, repo := gitee.GetOwnerAndRepoByEvent(e.NoteEvent)
	number := e.GetIssueNumber()
	action := &issueNoteAction{l.ghc, org, repo, number}
	repoLabels, issueLabels, err := l.getRepoAndCommentOBJLabels(action, org, repo)
	if err != nil {
		return err
	}
	if len(rmMatches) > 0 {
		removeMatchLabels(rmMatches, issueLabels, action, log)
	}
	if len(addMatches) > 0 {
		return addMatchLabels(addMatches, issueLabels, repoLabels, action, log)
	}
	return nil
}

func (l *label) getRepoAndCommentOBJLabels(action noteEventAction, org, repo string) (map[string]string, map[string]string, error) {
	repoLabels, err := l.ghc.GetRepoLabels(org, repo)
	if err != nil {
		return nil, nil, err
	}
	objLabels, err := action.getAllLabels()
	if err != nil {
		return nil, nil, err
	}
	repoLabelsMap := labelsTransformMap(repoLabels)
	objLabelsMap := labelsTransformMap(objLabels)
	return repoLabelsMap, objLabelsMap, nil
}

func labelsTransformMap(labels []sdk.Label) map[string]string {
	lm := make(map[string]string, len(labels))
	for _, v := range labels {
		k := strings.ToLower(v.Name)
		lm[k] = v.Name
	}
	return lm
}

func removeMatchLabels(matches [][]string, labels map[string]string, action noteEventAction, log *logrus.Entry) {
	labelsToRemove := getLabelsFromREMatches(matches)

	// Remove labels
	for _, labelToRemove := range labelsToRemove {
		if label, ok := labels[labelToRemove]; ok {
			if err := action.removeLabel(label); err != nil {
				log.WithError(err).Errorf("Gitee failed to add the following label: %s", label)
			}
		}
	}
}

func addMatchLabels(matches [][]string, currentLabels, repoLabels map[string]string, action noteEventAction, log *logrus.Entry) error {
	labelsToAdd := getLabelsFromREMatches(matches)

	var noSuchLabelsInRepo []string
	// Add labels
	var canAddLabel []string
	for _, labelToAdd := range labelsToAdd {
		if _, ok := currentLabels[labelToAdd]; ok {
			continue
		}

		if label, ok := repoLabels[labelToAdd]; !ok {
			noSuchLabelsInRepo = append(noSuchLabelsInRepo, labelToAdd)
		} else {
			canAddLabel = append(canAddLabel, label)
		}
	}

	if len(canAddLabel) > 0 {
		if err := action.addLabel(canAddLabel); err != nil {
			return err
		}
	}

	if len(noSuchLabelsInRepo) == 0 {
		return nil
	}
	msg := fmt.Sprintf(
		"The label(s) `%s` cannot be applied, because the repository doesn't have them",
		strings.Join(noSuchLabelsInRepo, ", "),
	)
	return action.addComment(msg)

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
		return fmt.Sprintf("The label plugin will work on %s and %s labels.",
			strings.Join(formattedLabels[:len(formattedLabels)-1], ", "), formattedLabels[len(formattedLabels)-1])
	}
	return ""
}
