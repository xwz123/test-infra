package label

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
)

type noteEventAction interface {
	addLabel(label []string) error
	addComment(comment string) error
	removeLabel(label string) error
	getAllLabels() ([]sdk.Label, error)
}

type issueNoteAction struct {
	client giteeClient
	org    string
	repo   string
	number string
}

func (ia *issueNoteAction) addLabel(label []string) error {
	return ia.client.AddMultiIssueLabel(ia.org, ia.repo, ia.number, label)
}

func (ia *issueNoteAction) addComment(comment string) error {
	return ia.client.CreateIssueComment(ia.org, ia.repo, ia.number, comment)
}

func (ia *issueNoteAction) removeLabel(label string) error {
	return ia.client.RemoveIssueLabel(ia.org, ia.repo, ia.number, label)
}

func (ia *issueNoteAction) getAllLabels() ([]sdk.Label, error) {
	return ia.client.GetIssueLabels(ia.org, ia.repo, ia.number)
}

type prNoteAction struct {
	client giteeClient
	org    string
	repo   string
	number int
}

func (pa *prNoteAction) addLabel(label []string) error {
	return pa.client.AddMultiPRLabel(pa.org, pa.repo, pa.number, label)
}

func (pa *prNoteAction) addComment(comment string) error {
	return pa.client.CreatePRComment(pa.org, pa.repo, pa.number, comment)
}

func (pa *prNoteAction) removeLabel(label string) error {
	return pa.client.RemovePRLabel(pa.org, pa.repo, pa.number, label)
}

func (pa *prNoteAction) getAllLabels() ([]sdk.Label, error) {
	return pa.client.GetPRLabels(pa.org, pa.repo, pa.number)
}
