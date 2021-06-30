package gitee

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/antihax/optional"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/github"
)

var _ Client = (*client)(nil)

type client struct {
	ac *sdk.APIClient

	mut      sync.Mutex // protects botName and email
	userData *sdk.User
}

func NewClient(getToken func() []byte) Client {
	token := string(getToken())

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	conf := sdk.NewConfiguration()
	conf.HTTPClient = oauth2.NewClient(context.Background(), ts)

	c := sdk.NewAPIClient(conf)
	return &client{ac: c}
}

// BotName returns the login of the authenticated identity.
func (c *client) BotName() (string, error) {
	err := c.getUserData()
	if err != nil {
		return "", err
	}

	return c.userData.Login, nil
}

func (c *client) Email() (string, error) {
	err := c.getUserData()
	if err != nil {
		return "", err
	}

	return c.userData.Email, nil
}

func (c *client) BotUser() (*github.User, error) {
	err := c.getUserData()
	if err != nil {
		return nil, err
	}

	d := c.userData
	u := github.User{
		Login: d.Login,
		Name:  d.Name,
		Email: d.Email,
		ID:    int(d.Id),
	}

	return &u, nil
}

func (c *client) CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (sdk.PullRequest, error) {
	opts := sdk.CreatePullRequestParam{
		Title:             title,
		Head:              head,
		Base:              base,
		Body:              body,
		PruneSourceBranch: true,
	}

	pr, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPulls(
		context.Background(), org, repo, opts)

	return pr, formatErr(err, "create pull request")
}

func (c *client) GetPullRequests(org, repo string, opts ListPullRequestOpt) ([]sdk.PullRequest, error) {

	setStr := func(t *optional.String, v string) {
		if v != "" {
			*t = optional.NewString(v)
		}
	}

	opt := sdk.GetV5ReposOwnerRepoPullsOpts{}
	setStr(&opt.State, opts.State)
	setStr(&opt.Head, opts.Head)
	setStr(&opt.Base, opts.Base)
	setStr(&opt.Sort, opts.Sort)
	setStr(&opt.Direction, opts.Direction)
	if opts.MilestoneNumber > 0 {
		opt.MilestoneNumber = optional.NewInt32(int32(opts.MilestoneNumber))
	}
	if opts.Labels != nil && len(opts.Labels) > 0 {
		opt.Labels = optional.NewString(strings.Join(opts.Labels, ","))
	}

	var r []sdk.PullRequest
	p := int32(1)
	for {
		opt.Page = optional.NewInt32(p)
		prs, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPulls(context.Background(), org, repo, &opt)
		if err != nil {
			return nil, formatErr(err, "get pull requests")
		}

		if len(prs) == 0 {
			break
		}

		r = append(r, prs...)
		p++
	}

	return r, nil
}

func (c *client) UpdatePullRequest(org, repo string, number int32, param sdk.PullRequestUpdateParam) (sdk.PullRequest, error) {
	pr, _, err := c.ac.PullRequestsApi.PatchV5ReposOwnerRepoPullsNumber(context.Background(), org, repo, number, param)
	return pr, formatErr(err, "update pull request")
}

func (c *client) GetGiteePullRequest(org, repo string, number int) (sdk.PullRequest, error) {
	pr, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumber(
		context.Background(), org, repo, int32(number), nil)
	return pr, formatErr(err, "get pull request")
}

func (c *client) getUserData() error {
	if c.userData == nil {
		c.mut.Lock()
		defer c.mut.Unlock()

		if c.userData == nil {
			u, _, err := c.ac.UsersApi.GetV5User(context.Background(), nil)
			if err != nil {
				return formatErr(err, "fetch bot name")
			}
			c.userData = &u
		}
	}
	return nil
}

func (c *client) ListCollaborators(org, repo string) ([]github.User, error) {
	var r []github.User

	opt := sdk.GetV5ReposOwnerRepoCollaboratorsOpts{}
	p := int32(1)
	for {
		opt.Page = optional.NewInt32(p)
		cs, _, err := c.ac.RepositoriesApi.GetV5ReposOwnerRepoCollaborators(context.Background(), org, repo, &opt)
		if err != nil {
			return nil, formatErr(err, "list collaborators")
		}
		if len(cs) == 0 {
			break
		}
		for _, i := range cs {
			c := github.User{
				Login: i.Login,
			}
			r = append(r, c)
		}
		p++
	}
	return r, nil
}

func (c *client) GetRef(org, repo, ref string) (string, error) {
	branch := strings.TrimPrefix(ref, "heads/")
	b, _, err := c.ac.RepositoriesApi.GetV5ReposOwnerRepoBranchesBranch(context.Background(), org, repo, branch, nil)
	if err != nil {
		return "", formatErr(err, "get branch")
	}

	return b.Commit.Sha, nil
}

func (c *client) GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error) {
	fs, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumberFiles(
		context.Background(), org, repo, int32(number), nil)
	if err != nil {
		return nil, formatErr(err, "list files of pr")
	}

	var r []github.PullRequestChange

	for _, f := range fs {
		r = append(r, github.PullRequestChange{Filename: f.Filename})
	}
	return r, nil
}

func (c *client) GetPRLabels(org, repo string, number int) ([]sdk.Label, error) {
	var r []sdk.Label

	p := int32(1)
	opt := sdk.GetV5ReposOwnerRepoPullsNumberLabelsOpts{}
	for {
		opt.Page = optional.NewInt32(p)
		ls, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumberLabels(
			context.Background(), org, repo, int32(number), &opt)
		if err != nil {
			return nil, formatErr(err, "list labels of pr")
		}

		if len(ls) == 0 {
			break
		}

		r = append(r, ls...)
		p++
	}

	return r, nil
}

func (c *client) ListPRComments(org, repo string, number int) ([]sdk.PullRequestComments, error) {
	var r []sdk.PullRequestComments

	p := int32(1)
	opt := sdk.GetV5ReposOwnerRepoPullsNumberCommentsOpts{}
	for {
		opt.Page = optional.NewInt32(p)
		cs, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumberComments(
			context.Background(), org, repo, int32(number), &opt)
		if err != nil {
			return nil, formatErr(err, "list comments of pr")
		}

		if len(cs) == 0 {
			break
		}

		r = append(r, cs...)
		p++
	}

	return r, nil
}

func (c *client) ListPrIssues(org, repo string, number int32) ([]sdk.Issue, error) {
	var issues []sdk.Issue
	p := int32(1)
	opt := sdk.GetV5ReposOwnerRepoPullsNumberIssuesOpts{}
	for {
		opt.Page = optional.NewInt32(p)
		iss, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumberIssues(context.Background(), org, repo, number, &opt)
		if err != nil {
			return nil, formatErr(err, "list issues of pr")
		}
		if len(iss) == 0 {
			break
		}
		issues = append(issues, iss...)
		p++
	}
	return issues, nil
}

func (c *client) DeletePRComment(org, repo string, ID int) error {
	_, err := c.ac.PullRequestsApi.DeleteV5ReposOwnerRepoPullsCommentsId(
		context.Background(), org, repo, int32(ID), nil)
	return formatErr(err, "delete comment of pr")
}

func (c *client) CreatePRComment(org, repo string, number int, comment string) error {
	opt := sdk.PullRequestCommentPostParam{Body: comment}
	_, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPullsNumberComments(
		context.Background(), org, repo, int32(number), opt)
	return formatErr(err, "create comment of pr")
}

func (c *client) UpdatePRComment(org, repo string, commentID int, comment string) error {
	opt := sdk.PullRequestCommentPatchParam{Body: comment}
	_, _, err := c.ac.PullRequestsApi.PatchV5ReposOwnerRepoPullsCommentsId(
		context.Background(), org, repo, int32(commentID), opt)
	return formatErr(err, "update comment of pr")
}

func (c *client) AddPRLabel(org, repo string, number int, label string) error {
	opt := sdk.PullRequestLabelPostParam{Body: []string{label}}
	_, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPullsNumberLabels(
		context.Background(), org, repo, int32(number), opt)
	return formatErr(err, "add label for pr")
}

func (c *client) AddMultiPRLabel(org, repo string, number int, label []string) error {
	opt := sdk.PullRequestLabelPostParam{Body: label}
	_, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPullsNumberLabels(
		context.Background(), org, repo, int32(number), opt)
	return formatErr(err, "add multi label for pr")
}

func (c *client) RemovePRLabel(org, repo string, number int, label string) error {
	// gitee's bug, it can't deal with the label which includes '/'
	label = strings.Replace(label, "/", "%2F", -1)

	v, err := c.ac.PullRequestsApi.DeleteV5ReposOwnerRepoPullsLabel(
		context.Background(), org, repo, int32(number), label, nil)

	if v != nil && v.StatusCode == 404 {
		return nil
	}
	return formatErr(err, "remove label of pr")
}

func (c *client) ClosePR(org, repo string, number int) error {
	opt := sdk.PullRequestUpdateParam{State: StatusClosed}
	_, err := c.UpdatePullRequest(org, repo, int32(number), opt)
	return formatErr(err, "close pr")
}

func (c *client) AssignPR(org, repo string, number int, logins []string) error {
	opt := sdk.PullRequestAssigneePostParam{Assignees: strings.Join(logins, ",")}
	_, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPullsNumberAssignees(
		context.Background(), org, repo, int32(number), opt)
	return formatErr(err, "assign reviewer to pr")
}

func (c *client) UnassignPR(org, repo string, number int, logins []string) error {
	_, _, err := c.ac.PullRequestsApi.DeleteV5ReposOwnerRepoPullsNumberAssignees(
		context.Background(), org, repo, int32(number), strings.Join(logins, ","), nil)
	return formatErr(err, "unassign reviewer from pr")
}

func (c *client) GetPRCommits(org, repo string, number int) ([]sdk.PullRequestCommits, error) {
	commits, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPullsNumberCommits(
		context.Background(), org, repo, int32(number), nil)
	return commits, formatErr(err, "get pr commits")
}

func (c *client) AssignGiteeIssue(org, repo string, number string, login string) error {
	opt := sdk.IssueUpdateParam{Repo: repo, Assignee: login}
	_, v, err := c.ac.IssuesApi.PatchV5ReposOwnerIssuesNumber(
		context.Background(), org, number, opt)

	if err != nil {
		if v.StatusCode == 403 {
			return ErrorForbidden{err: formatErr(err, "assign assignee to issue").Error()}
		}
	}
	return formatErr(err, "assign assignee to issue")
}

func (c *client) UnassignGiteeIssue(org, repo string, number string, login string) error {
	return c.AssignGiteeIssue(org, repo, number, " ")
}

func (c *client) CreateIssueComment(org, repo string, number string, comment string) error {
	opt := sdk.IssueCommentPostParam{Body: comment}
	_, _, err := c.ac.IssuesApi.PostV5ReposOwnerRepoIssuesNumberComments(
		context.Background(), org, repo, number, opt)
	return formatErr(err, "create issue comment")
}

func (c *client) IsCollaborator(owner, repo, login string) (bool, error) {
	v, err := c.ac.RepositoriesApi.GetV5ReposOwnerRepoCollaboratorsUsername(
		context.Background(), owner, repo, login, nil)
	if err != nil {
		if v.StatusCode == 404 {
			return false, nil
		}
		return false, formatErr(err, "get collaborator of pr")
	}
	return true, nil
}

func (c *client) IsMember(org, login string) (bool, error) {
	_, v, err := c.ac.OrganizationsApi.GetV5OrgsOrgMembershipsUsername(
		context.Background(), org, login, nil)
	if err != nil {
		if v.StatusCode == 404 {
			return false, nil
		}
		return false, formatErr(err, "get member of org")
	}
	return true, nil
}

func (c *client) GetPRCommit(org, repo, SHA string) (sdk.RepoCommit, error) {
	v, _, err := c.ac.RepositoriesApi.GetV5ReposOwnerRepoCommitsSha(
		context.Background(), org, repo, SHA, nil)
	if err != nil {
		return v, formatErr(err, "get commit info")
	}

	return v, nil
}

func (c *client) GetSingleCommit(org, repo, SHA string) (github.SingleCommit, error) {
	var r github.SingleCommit

	v, _, err := c.ac.RepositoriesApi.GetV5ReposOwnerRepoCommitsSha(
		context.Background(), org, repo, SHA, nil)
	if err != nil {
		return r, formatErr(err, "get commit info")
	}

	r.Commit.Tree.SHA = v.Commit.Tree.Sha
	return r, nil
}

func (c *client) MergePR(owner, repo string, number int, opt sdk.PullRequestMergePutParam) error {
	_, err := c.ac.PullRequestsApi.PutV5ReposOwnerRepoPullsNumberMerge(
		context.Background(), owner, repo, int32(number), opt)
	return formatErr(err, "merge pr")
}

func (c *client) GetRepos(org string) ([]sdk.Project, error) {
	opt := sdk.GetV5OrgsOrgReposOpts{}
	var r []sdk.Project
	p := int32(1)
	for {
		opt.Page = optional.NewInt32(p)
		ps, _, err := c.ac.RepositoriesApi.GetV5OrgsOrgRepos(context.Background(), org, &opt)
		if err != nil {
			return nil, formatErr(err, "list repos")
		}

		if len(ps) == 0 {
			break
		}
		r = append(r, ps...)
		p++
	}

	return r, nil
}

func (c *client) GetRepoLabels(owner, repo string) ([]sdk.Label, error) {
	labels, _, err := c.ac.LabelsApi.GetV5ReposOwnerRepoLabels(context.Background(), owner, repo, nil)
	return labels, formatErr(err, "get repo labels")
}

func (c *client) AddIssueLabel(org, repo, number, label string) error {
	opt := sdk.PullRequestLabelPostParam{Body: []string{label}}
	_, _, err := c.ac.LabelsApi.PostV5ReposOwnerRepoIssuesNumberLabels(
		context.Background(), org, repo, number, opt)
	return formatErr(err, "add issue label")
}

func (c *client) AddMultiIssueLabel(org, repo, number string, label []string) error {
	opt := sdk.PullRequestLabelPostParam{Body: label}
	_, _, err := c.ac.LabelsApi.PostV5ReposOwnerRepoIssuesNumberLabels(
		context.Background(), org, repo, number, opt)
	return formatErr(err, "add issue label")
}

func (c *client) RemoveIssueLabel(org, repo, number, label string) error {
	label = strings.Replace(label, "/", "%2F", -1)
	_, err := c.ac.LabelsApi.DeleteV5ReposOwnerRepoIssuesNumberLabelsName(
		context.Background(), org, repo, number, label, nil)
	return formatErr(err, "rm issue label")
}

func (c *client) ReplacePRAllLabels(owner, repo string, number int, labels []string) error {
	opt := sdk.PullRequestLabelPostParam{Body: labels}
	_, _, err := c.ac.PullRequestsApi.PutV5ReposOwnerRepoPullsNumberLabels(context.Background(), owner, repo, int32(number), opt)
	return formatErr(err, "replace pr labels")
}

func (c *client) CloseIssue(owner, repo string, number string) error {
	opt := sdk.IssueUpdateParam{Repo: repo, State: StatusClosed}
	_, err := c.UpdateIssue(owner, number, opt)
	return formatErr(err, "close issue")
}

func (c *client) ReopenIssue(owner, repo string, number string) error {
	opt := sdk.IssueUpdateParam{Repo: repo, State: StatusOpen}
	_, err := c.UpdateIssue(owner, number, opt)
	return formatErr(err, "reopen issue")
}

func (c *client) UpdateIssue(owner, number string, param sdk.IssueUpdateParam) (sdk.Issue, error) {
	issue, _, err := c.ac.IssuesApi.PatchV5ReposOwnerIssuesNumber(context.Background(), owner, number, param)
	return issue, formatErr(err, "update issue")
}

func (c *client) GetIssueLabels(org, repo, number string) ([]sdk.Label, error) {
	labels, _, err := c.ac.LabelsApi.GetV5ReposOwnerRepoIssuesNumberLabels(context.Background(), org, repo, number, nil)
	return labels, formatErr(err, "get issue labels")
}

func (c *client) GetIssue(org, repo, number string) (sdk.Issue, error) {
	issue, _, err := c.ac.IssuesApi.GetV5ReposOwnerRepoIssuesNumber(context.Background(), org, repo, number, nil)
	return issue, formatErr(err, "get issue")
}

func formatErr(err error, doWhat string) error {
	if err == nil {
		return err
	}

	return fmt.Errorf("Failed to %s: %s", doWhat, err.Error())
}
