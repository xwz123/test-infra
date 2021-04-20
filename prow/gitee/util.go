package gitee

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
)

//GetOwnerAndRepoByEvent obtain the owner and repository name from the event
func GetOwnerAndRepoByEvent(e interface{}) (string, string) {
	var repository *sdk.ProjectHook

	switch t := e.(type) {
	case *sdk.PullRequestEvent:
		repository = t.Repository
	case *sdk.NoteEvent:
		repository = t.Repository
	case *sdk.IssueEvent:
		repository = t.Repository
	case *sdk.PushEvent:
		repository = t.Repository
	default:
		return "", ""
	}

	return repository.Namespace, repository.Path
}
//GetOwnerAndRepoByPRBranch obtain the owner and repository name from the pullrequest's branch
func GetOwnerAndRepoByPRBranch(prb *sdk.BranchHook) (string, string) {
	if prb.Repo == nil {
		return "", ""
	}
	return prb.Repo.Namespace, prb.Repo.Path
}
