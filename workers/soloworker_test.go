package workers_test

import (
	"io/ioutil"
	"os"
	"testing"
	workers "yeet/workers"

	"gopkg.in/yaml.v3"
)

type SoloWorker_Test_Config struct {
	Test1 struct {
		CodePath string `yaml:"codepath"`
		MinSize  int    `yaml:"min_size"`
		NumRepos int    `yaml:"num_repos"`
	}
}

func loadSoloWorker_config() *SoloWorker_Test_Config {
	yamlFile, err := ioutil.ReadFile("soloworker_test.yaml")
	if err != nil {
		panic(err)
	}

	var config SoloWorker_Test_Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

func TestSoloWorker1(t *testing.T) {
	config := loadSoloWorker_config().Test1
	w := workers.SoloWorker{"test.json", config.CodePath}
	err := w.Refresh()
	if err != nil {
		t.Fatalf(`Error refreshing repo list: %v`, err)
	}
	fi, err := os.Stat("test.json")
	if err != nil || fi.Size() < int64(config.MinSize) {
		t.Fatalf(`Error saving repo list file: %v`, err)
	}
	r, err := w.GetList()
	if err != nil || len(r.RepoList) != config.NumRepos {
		t.Fatalf(`Error loading repo list file: %v`, err)
	}
}
