package label

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	sdk "gitee.com/openeuler/go-gitee/gitee"
)

type handler interface {
	addLabel(label []string) error
	addComment(comment string) error
	removeLabel(label string) error
	getLabels() (map[string]string, error)
	handleComment() error
}

type noteHandle struct {
	client       giteeClient
	log          *logrus.Entry
	addLabels    []string
	removeLabels []string
	org          string
	repo         string
}

func (nh *noteHandle) getRepoLabels() (map[string]string, error) {
	labels, err := nh.client.GetRepoLabels(nh.org, nh.repo)
	if err != nil {
		return nil, err
	}
	return labelsTransformMap(labels), nil
}

type issueNoteHandle struct {
	noteHandle
	number string
}

func (inh *issueNoteHandle) addLabel(label []string) error {
	return inh.client.AddMultiIssueLabel(inh.org, inh.repo, inh.number, label)
}

func (inh *issueNoteHandle) addComment(comment string) error {
	return inh.client.CreateIssueComment(inh.org, inh.repo, inh.number, comment)
}

func (inh *issueNoteHandle) removeLabel(label string) error {
	return inh.client.RemoveIssueLabel(inh.org, inh.repo, inh.number, label)
}

func (inh *issueNoteHandle) getLabels() (map[string]string, error) {
	labels, err := inh.client.GetIssueLabels(inh.org, inh.repo, inh.number)
	if err != nil {
		return nil, err
	}
	return labelsTransformMap(labels), nil
}

func (inh *issueNoteHandle) handleComment() error {
	repoLabels, err := inh.getRepoLabels()
	if err != nil {
		return err
	}
	issueLabels, err := inh.getLabels()
	if err != nil {
		return err
	}

	if len(inh.removeLabels) > 0 {
		removeLabels(inh, issueLabels, inh.removeLabels, inh.log)
	}

	if len(inh.addLabels) > 0 {
		return addLabels(inh, issueLabels, repoLabels, inh.addLabels)
	}
	return nil
}

type prNoteHandle struct {
	noteHandle
	number int
}

func (pnh *prNoteHandle) addLabel(label []string) error {
	return pnh.client.AddMultiPRLabel(pnh.org, pnh.repo, pnh.number, label)
}

func (pnh *prNoteHandle) addComment(comment string) error {
	return pnh.client.CreatePRComment(pnh.org, pnh.repo, pnh.number, comment)
}

func (pnh *prNoteHandle) removeLabel(label string) error {
	return pnh.client.RemovePRLabel(pnh.org, pnh.repo, pnh.number, label)
}

func (pnh *prNoteHandle) getLabels() (map[string]string, error) {
	labels, err := pnh.client.GetPRLabels(pnh.org, pnh.repo, pnh.number)
	if err != nil {
		return nil, err
	}
	return labelsTransformMap(labels), nil
}

func (pnh *prNoteHandle) handleComment() error {
	repoLabels, err := pnh.getRepoLabels()
	if err != nil {
		return err
	}
	prLabels, err := pnh.getLabels()
	if err != nil {
		return err
	}

	if len(pnh.removeLabels) > 0 {
		removeLabels(pnh, prLabels, pnh.removeLabels, pnh.log)
	}

	if len(pnh.addLabels) > 0 {
		return addLabels(pnh, prLabels, repoLabels, pnh.addLabels)
	}

	return nil
}

func removeLabels(handle handler, currentLabels map[string]string, rmLabels []string, log *logrus.Entry) {
	for _, rmLabel := range rmLabels {
		if label, ok := currentLabels[rmLabel]; ok {
			if err := handle.removeLabel(label); err != nil {
				log.WithError(err).Errorf("Gitee failed to add the following label: %s", label)
			}
		}
	}
}

func addLabels(handle handler, curLabels, repoLabels map[string]string, labelsToAdd []string) error {
	var noSuchLabelsInRepo []string
	var canAddLabel []string
	for _, labelToAdd := range labelsToAdd {
		if _, ok := curLabels[labelToAdd]; ok {
			continue
		}
		if label, ok := repoLabels[labelToAdd]; !ok {
			noSuchLabelsInRepo = append(noSuchLabelsInRepo, labelToAdd)
		} else {
			canAddLabel = append(canAddLabel, label)
		}
	}
	if len(canAddLabel) > 0 {
		if err := handle.addLabel(canAddLabel); err != nil {
			return err
		}
	}

	if len(noSuchLabelsInRepo) > 0 {
		msg := fmt.Sprintf(
			"The label(s) `%s` cannot be applied, because the repository doesn't have them",
			strings.Join(noSuchLabelsInRepo, ", "),
		)
		return handle.addComment(msg)
	}

	return nil
}

func labelsTransformMap(labels []sdk.Label) map[string]string {
	lm := make(map[string]string, len(labels))
	for _, v := range labels {
		k := strings.ToLower(v.Name)
		lm[k] = v.Name
	}
	return lm
}
