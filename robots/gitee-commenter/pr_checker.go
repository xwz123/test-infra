package main

import (
	"github.com/sirupsen/logrus"
)

type prChecker struct {
	log    *logrus.Entry
	client commenterClient

	cItem  checkItem
	owner  string
	repo   string
	number int32
}

func (pc *prChecker) check() {
	//Get the details again and ensure that the status is up to date as much as possible
	pr, err := pc.client.GetGiteePullRequest(pc.owner, pc.repo, int(pc.number))
	if err != nil {
		pc.log.Error(err)
		return
	}
	if !needCheckPr(pr, pc.cItem) {
		return
	}
	if err := pc.client.CreatePRComment(pc.owner, pc.repo, int(pc.number), pc.cItem.Comment); err != nil {
		pc.log.Error(err)
	}

}
