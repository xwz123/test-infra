package gitee

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
)

//GetOwnerAndRepoByPREvent obtain the owner and repository name from the pullrequest's event
func GetOwnerAndRepoByPREvent(pre *sdk.PullRequestEvent) (string, string) {
	return pre.Repository.Namespace, pre.Repository.Path
}

//GetOwnerAndRepoByIssueEvent obtain the owner and repository name from the issue's event
func GetOwnerAndRepoByIssueEvent(pre *sdk.PullRequestEvent) (string, string) {
	return pre.Repository.Namespace, pre.Repository.Path
}
