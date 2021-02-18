package lifecycle

import (
	"regexp"
	"strings"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pluginhelp"
)

var (
	lifecycleLabels = []string{labels.LifecycleActive, labels.LifecycleFrozen, labels.LifecycleStale, labels.LifecycleRotten}
	lifecycleRe     = regexp.MustCompile(`(?mi)^/(remove-)?lifecycle (active|frozen|stale|rotten)\s*$`)
)

type lifecycle struct {
	fGpc plugins.GetPluginConfig
	gec  gitee.Client
}

type lifecycleClient interface {
	AddPRLabel(org, repo string, number int, label string) error
	RemovePRLabel(org, repo string, number int, label string) error
	GetPRLabels(org, repo string, number int) ([]sdk.Label, error)
	AddIssueLabel(org, repo, number, label string) error
	RemoveIssueLabel(org, repo, number, label string) error
	GetIssueLabels(org, repo, number string) ([]sdk.Label, error)
}

func NewLifeCycle(f plugins.GetPluginConfig, gec gitee.Client) plugins.Plugin {
	return &lifecycle{
		fGpc: f,
		gec:  gec,
	}
}

func (l *lifecycle) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "Close an issue or PR",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/close",
		Featured:    false,
		Description: "Closes an issue or PullRequest.",
		Examples:    []string{"/close"},
		WhoCanUse:   "Authors and collaborators on the repository can trigger this command.",
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/reopen",
		Description: "Reopens an issue ",
		Featured:    false,
		WhoCanUse:   "Authors and collaborators on the repository can trigger this command.",
		Examples:    []string{"/reopen"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/[remove-]lifecycle <frozen|stale|rotten>",
		Description: "Flags an issue or PR as frozen/stale/rotten",
		Featured:    false,
		WhoCanUse:   "Anyone can trigger this command.",
		Examples:    []string{"/lifecycle frozen", "/remove-lifecycle stale"},
	})
	return pluginHelp, nil
}

func (l *lifecycle) PluginName() string {
	return "lifecycle"
}

func (l *lifecycle) NewPluginConfig() plugins.PluginConfig {
	return nil
}

func (l *lifecycle) RegisterEventHandler(p plugins.Plugins) {
	name := l.PluginName()
	p.RegisterNoteEventHandler(name, l.handleNoteEvent)
}

func (l *lifecycle) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handleNoteEvent")
	}()

	eType := *(e.NoteableType)
	if *(e.Action) != "comment" || (!isPr(eType) && eType != "Issue") {
		log.Debug("Event is not a creation of a comment for PR or issue, skipping.")
		return nil
	}
	if err := handleReopen(l.gec, log, e); err != nil {
		return err
	}
	if err := handleClose(l.gec, log, e); err != nil {
		return err
	}
	return handle(l.gec, log, e)
}

func isPr(et string) bool {
	return et == "PullRequest"
}

func handle(gc lifecycleClient, log *logrus.Entry, e *sdk.NoteEvent) error {
	// Only consider new comments.
	if *e.Action != "comment" {
		return nil
	}
	for _, mat := range lifecycleRe.FindAllStringSubmatch(e.Comment.Body, -1) {
		if err := handleOne(gc, log, e, mat); err != nil {
			return err
		}
	}
	return nil
}

func handleOne(gc lifecycleClient, log *logrus.Entry, e *sdk.NoteEvent, mat []string) error {
	isPr := isPr(*e.NoteableType)
	remove := mat[1] != ""
	cmd := mat[2]
	lbl := "lifecycle/" + cmd
	lbs, err := getLabels(gc, e, isPr)
	if err != nil {
		log.WithError(err).Errorf("Failed to get labels.")
	}
	if hasLabel(lbl, lbs) && remove {
		return removeLabel(gc, e, isPr, lbl)
	}
	if !hasLabel(lbl, lbs) && !remove {
		for _, label := range lifecycleLabels {
			if label != lbl && hasLabel(label, lbs) {
				if err := removeLabel(gc, e, isPr, label); err != nil {
					log.WithError(err).Errorf("Gitee failed to remove the following label: %s", label)
				}
			}
		}
		if err := addLabel(gc, e, isPr, lbl); err != nil {
			log.WithError(err).Errorf("Gitee failed to add the following label: %s", lbl)
		}
	}
	return nil
}

func removeLabel(gc lifecycleClient, e *sdk.NoteEvent, pr bool, lbl string) error {
	if pr {
		return gc.RemovePRLabel(e.Repository.Namespace, e.Repository.Path, int(e.PullRequest.Number), lbl)
	}
	return gc.RemoveIssueLabel(e.Repository.Namespace, e.Repository.Path, e.Issue.Number, lbl)
}

func getLabels(gc lifecycleClient, e *sdk.NoteEvent, isPr bool) ([]sdk.Label, error) {
	if isPr {
		return gc.GetPRLabels(e.Repository.Namespace, e.Repository.Path, int(e.PullRequest.Number))
	}
	return gc.GetIssueLabels(e.Repository.Namespace, e.Repository.Path, e.Issue.Number)
}

func addLabel(gc lifecycleClient, e *sdk.NoteEvent, pr bool, lbl string) error {
	if pr {
		return gc.AddPRLabel(e.Repository.Namespace, e.Repository.Path, int(e.PullRequest.Number), lbl)
	}
	return gc.AddIssueLabel(e.Repository.Namespace, e.Repository.Path, e.Issue.Number, lbl)
}

func hasLabel(label string, issueLabels []sdk.Label) bool {
	for _, l := range issueLabels {
		if strings.ToLower(l.Name) == strings.ToLower(label) {
			return true
		}
	}
	return false
}
