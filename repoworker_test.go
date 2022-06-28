package main

import (
	"io/ioutil"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

type RepoWorker_Test_Config struct {
	Test1 struct {
		SampleRepoPath string   `yaml:"samplerepopath"`
		SampleRepoName string   `yaml:"samplereponame"`
		Remotes        []string `yaml:"remotes,flow"`
	}
}

func loadRepoWorker_config() *RepoWorker_Test_Config {
	yamlFile, err := ioutil.ReadFile("repoworker_test.yaml")
	if err != nil {
		panic(err)
	}

	var config RepoWorker_Test_Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

func TestRepoWorker1(t *testing.T) {
	config := loadRepoWorker_config().Test1
	repoInfo := RepoInfo{config.SampleRepoPath, config.SampleRepoName}
	init := RepoWorkerInitializer{&repoInfo}
	rw := init.NewRepoWorker()
	if rw.Branch == "" {
		t.Fatalf(`Repo worker has no branch`)
	}
	if !reflect.DeepEqual(rw.Remotes, config.Remotes) {
		t.Fatalf(`Repo worker has remote name mismatch`)
	}
}
