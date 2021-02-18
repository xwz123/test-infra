package main

import (
	"strings"
	"sync"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/gitee"
)

type empty = struct{}

type commenterClient interface {
	GetRepos(org string) ([]sdk.Project, error)
	GetIssues(org, repo string, opts gitee.ListIssueOpt) ([]sdk.Issue, error)
	GetIssue(org, repo, number string) (sdk.Issue, error)
	CreateGiteeIssueComment(org, repo string, number string, comment string) error
	GetPullRequests(org, repo string, opts gitee.ListPullRequestOpt) ([]sdk.PullRequest, error)
	GetGiteePullRequest(org, repo string, number int) (sdk.PullRequest, error)
	CreatePRComment(org, repo string, number int, comment string) error
}

type orgRepo struct {
	org  string
	repo string
}

type commenter struct {
	log  *logrus.Entry
	pool *checkerPool
	cfg  *configs

	gcc          commenterClient
	maxGoroutine int
	wg           sync.WaitGroup
}

func (c *commenter) run() {
	defer c.shutdown()
	repoMap := c.getRepoByCfg()
	if len(repoMap) == 0 {
		c.log.Infoln("no repo to check.")
		return
	}
	max := c.maxGoroutine
	var lc []orgRepo
	for v := range repoMap {
		org, repo := splitToOrgRepo(v)
		if org == "" {
			continue
		}
		lc = append(lc, orgRepo{org: org, repo: repo})
	}
	if len(lc) < max {
		max = len(lc)
	}
	c.wg.Add(max)
	handleRepo(max, lc, func(rg orgRepo) {
		if !c.cfg.UncheckIssue {
			c.genIssueChecker(rg)
		}
		if !c.cfg.UncheckPr {
			c.genPrChecker(rg)
		}
	}, func() {
		c.wg.Done()
	})
}

func (c *commenter) shutdown() {
	c.wg.Wait()
	c.pool.shutdown()
}

func (c *commenter) getRepoByCfg() map[string]empty {
	mRepo := make(map[string]empty)
	if len(c.cfg.Orgs) > 0 {
		for _, v := range c.cfg.Orgs {
			rs, err := c.gcc.GetRepos(v)
			if err != nil {
				c.log.Error(err)
				continue
			}
			for _, r := range rs {
				if r.FullName == "" {
					continue
				}
				mRepo[r.FullName] = empty{}
			}
		}
	}
	if len(c.cfg.Repos) > 0 {
		for _, v := range c.cfg.Repos {
			if _, ok := mRepo[v]; !ok {
				mRepo[v] = empty{}
			}
		}
	}
	if len(c.cfg.ExcludeRepos) > 0 {
		for _, v := range c.cfg.ExcludeRepos {
			if _, ok := mRepo[v]; ok {
				delete(mRepo, v)
			}
		}
	}
	return mRepo
}

func (c *commenter) genIssueChecker(or orgRepo) {
	opt := gitee.ListIssueOpt{State: "open"}
	iss, err := c.gcc.GetIssues(or.org, or.repo, opt)
	if err != nil {
		c.log.Error(err, or)
		return
	}
	count := len(iss)
	if count == 0 {
		return
	}
	//Generate corresponding inspection tasks based on configuration
	for _, v := range iss {
		for _, ck := range c.cfg.Checks {
			if needCheckIssue(v, ck) {
				pc := issueChecker{log: c.log, client: c.gcc, cItem: ck,
					number: v.Number, org: or.org, repo: or.repo}
				c.pool.run(&pc)
			}

		}
	}
}

func (c *commenter) genPrChecker(or orgRepo) {
	opt := gitee.ListPullRequestOpt{State: "open"}
	prs, err := c.gcc.GetPullRequests(or.org, or.repo, opt)
	if err != nil {
		c.log.Error(err, or)
		return
	}
	count := len(prs)
	if count == 0 {
		return
	}
	//Generate corresponding inspection tasks based on configuration
	for _, v := range prs {
		for _, ck := range c.cfg.Checks {
			if needCheckPr(v, ck) {
				pc := prChecker{log: c.log, client: c.gcc, cItem: ck, owner: or.org, repo: or.repo, number: v.Number}
				c.pool.run(&pc)
			}
		}
	}
}

func newCommenter(clt commenterClient, cfg *configs, log *logrus.Entry, maxGoroutine int) *commenter {
	co := commenter{gcc: clt, cfg: cfg, log: log, maxGoroutine: maxGoroutine}
	co.pool = newCheckerPool(maxGoroutine)
	return &co
}

func splitToOrgRepo(s string) (string, string) {
	v := strings.Split(s, "/")
	if len(v) == 2 {
		return v[0], v[1]
	}
	return "", ""
}

func handleRepo(mg int, rgs []orgRepo, proc func(repo orgRepo), done func()) {
	qu := make(chan orgRepo, len(rgs))
	for _, v := range rgs {
		qu <- v
	}
	close(qu)
	for i := 0; i < mg; i++ {
		go func() {
			defer done()
			for gr := range qu {
				proc(gr)
			}
		}()
	}
}

func needCheckIssue(issue sdk.Issue, ci checkItem) bool {
	if time.Now().Sub(issue.UpdatedAt) <= ci.Updated.Duration {
		return false
	}
	mla := make(map[string]empty, len(issue.Labels))
	for _, v := range issue.Labels {
		mla[v.Name] = empty{}
	}
	for _, v := range ci.Labels {
		if _, ok := mla[v]; !ok {
			return false
		}
	}
	for _, v := range ci.ExcludeLabels {
		if _, ok := mla[v]; ok {
			return false
		}
	}
	return true
}

func needCheckPr(pr sdk.PullRequest, ci checkItem) bool {
	t, err := time.Parse(time.RFC3339, pr.UpdatedAt)
	if err != nil {
		return false
	}
	if time.Now().Sub(t) <= ci.Updated.Duration {
		return false
	}
	mla := make(map[string]empty, len(pr.Labels))
	for _, v := range pr.Labels {
		mla[v.Name] = empty{}
	}
	for _, v := range ci.Labels {
		if _, ok := mla[v]; !ok {
			return false
		}
	}
	for _, v := range ci.ExcludeLabels {
		if _, ok := mla[v]; ok {
			return false
		}
	}
	return true
}
