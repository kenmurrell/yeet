package workers_test

import (
	"io/ioutil"
	"reflect"
	"testing"
	workers "yeet/workers"

	"gopkg.in/yaml.v3"
)

type RepoWorker_Test_Config struct {
	Test1 struct {
		SampleRepoPath string   `yaml:"samplerepopath"`
		SampleRepoName string   `yaml:"samplereponame"`
		Remotes        []string `yaml:"remotes,flow"`
	}
}

func loadRepoWorker_config() (*RepoWorker_Test_Config, error) {
	yamlFile, err := ioutil.ReadFile("repoworker_test.yaml")
	if err != nil {
		return nil, err
	}

	var config RepoWorker_Test_Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func TestRepoWorker1(t *testing.T) {
	config, err := loadRepoWorker_config()
	if err != nil {
		t.Skip("No config file available, skipping")
	}
	config1 := config.Test1
	repoInfo := workers.RepoInfo{config1.SampleRepoPath, config1.SampleRepoName}
	init := workers.RepoWorkerInitializer{&repoInfo}
	rw := init.NewRepoWorker()
	if rw.Branch == "" {
		t.Fatalf(`Repo worker has no branch`)
	}
	if !reflect.DeepEqual(rw.Remotes, config1.Remotes) {
		t.Fatalf(`Repo worker has remote name mismatch`)
	}
}
