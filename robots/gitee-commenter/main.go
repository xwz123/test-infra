package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/logrusutil"
)

type options struct {
	config     string
	maxProcess int
	gitee      prowflagutil.GiteeOptions
	dryRun     bool
}

func (o *options) Validate() error {
	for _, group := range []flagutil.OptionGroup{&o.gitee} {
		if err := group.Validate(o.dryRun); err != nil {
			return err
		}
	}
	return nil
}

type configs struct {
	//Orgs organization to be checked
	Orgs []string `json:"orgs,omitempty"`
	//Repos the full path of the repo to be checked
	Repos []string `json:"repos,omitempty"`
	//Repos the full path of the repo to be unchecked
	ExcludeRepos []string `json:"exclude_repos,omitempty"`
	//UncheckPr Whether to check the PR in the repo
	UncheckPr bool `json:"uncheck_pr,omitempty"`
	//UncheckIssue Whether to check the issue in the repo
	UncheckIssue bool `json:"uncheck_issue,omitempty"`
	//Checks check task configuration
	Checks []checkItem `json:"checks"`
}

func (cfg configs) validate() error {
	if len(cfg.Orgs) == 0 && len(cfg.Repos) == 0 {
		return errors.New("No valid repo is configured ")
	}
	if len(cfg.Checks) == 0 {
		return errors.New("No valid check task is configured ")
	}
	for k := range cfg.Checks {
		if err := cfg.Checks[k].validate(); err != nil {
			return err
		}
	}
	return nil
}

type checkItem struct {
	//Labels the labels that the issue or PR should have.
	Labels []string `json:"labels,omitempty"`
	//ExcludeLabels  the labels that the issue or PR should not have.
	ExcludeLabels []string `json:"exclude_labels,omitempty"`
	//Updated How long has the issue or PR not updated
	Updated *metav1.Duration `json:"updated"`
	//Comment  content to be added comment when an issue or PR meets the filter criteria and is found
	Comment string `json:"comment"`
}

func (ci *checkItem) validate() error {
	if ci.Updated == nil {
		return errors.New("No valid updated is configured ")
		//ci.Updated = &metav1.Duration{Duration: 2 * time.Hour}
	}
	if ci.Comment == "" {
		return errors.New("No valid comment is configured ")
	}
	return nil
}

func flagOptions(fs *flag.FlagSet, args ...string) options {
	o := options{}
	fs.StringVar(&o.config, "config", "", "the config file path for check task")
	fs.IntVar(&o.maxProcess, "max-process", 10, "Number of threads in the thread pool,default 10.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	for _, group := range []flagutil.OptionGroup{&o.gitee} {
		group.AddFlags(fs)
	}
	_ = fs.Parse(args)
	return o
}

func loadConfig(path string) (*configs, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("prowConfig cannot be a dir - %s", path)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg configs
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	logrusutil.ComponentInit()
	o := flagOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.Fatal("Invalid options")
	}
	if o.config == "" {
		logrus.Fatal("empty -- config path")
	}
	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.gitee.TokenPath}); err != nil {
		logrus.Fatalf("Error starting secrets agent: %v", err)
	}

	config, err := loadConfig(o.config)
	if err != nil {
		logrus.WithError(err).Fatal("Error parse config file")
	}
	if err := config.validate(); err != nil {
		logrus.WithError(err).Fatal("config error")
	}
	gc, err := o.gitee.GiteeClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error init client.")
	}
	log := logrus.WithField("component", "gitee-commenter")
	comm := newCommenter(gc, config, log, o.maxProcess)
	comm.run()
}
