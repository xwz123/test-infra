package main

import (
	"github.com/sirupsen/logrus"
)

type issueChecker struct {
	log    *logrus.Entry
	client commenterClient
	cItem  checkItem

	org    string
	repo   string
	number string
}

func (ic *issueChecker) check() {
	//Get the details again and ensure that the status is up to date as much as possible
	issue, err := ic.client.GetIssue(ic.org, ic.repo, ic.number)
	if err != nil {
		ic.log.Error(err)
		return
	}
	if !needCheckIssue(issue, ic.cItem) {
		return
	}
	if err := ic.client.CreateGiteeIssueComment(ic.org, ic.repo, ic.number, ic.cItem.Comment); err != nil {
		ic.log.Error(err)
	}
}
