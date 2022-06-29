package main

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type ProgramConfig struct {
	MasterBranch string `yaml:"masterbranch"`
	FCRemote     string `yaml:"fcr"`
	RepoDir      string `yaml:"repodir"`
}

var config *ProgramConfig
var repolist *RepoList

var RepolistFilename string = "repolist.json"

func loadconfig() *ProgramConfig {
	var tempconfig ProgramConfig
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &tempconfig)
	if err != nil {
		panic(err)
	}
	return &tempconfig
}

func setup() {
	config = loadconfig()
	sw := SoloWorker{RepolistFilename, config.RepoDir}
	repolist, _ = sw.GetList()
}

func run(target string) {
	done := make(chan string)
	defer close(done)
	var n int = len(repolist.RepoList)
	for _, r := range repolist.RepoList {
		go workflow(target, r, done)
	}
	fmt.Printf("Started %d processes...\n", n)

	for i := 0; i < n; i++ {
		fmt.Println(<-done)
	}

}

func workflow(target string, r *RepoInfo, done chan<- string) {
	init := RepoWorkerInitializer{r}
	rw := init.NewRepoWorker()
	// Stash current changes on branch
	rw.Stash()
	// Update info from all remotes
	rw.Update()
	// Choose the correct remote
	remotes := rw.Remotes
	var remote string
	if len(remotes) == 1 {
		remote = remotes[0]
	} else if slices.Contains(remotes, config.FCRemote) {
		remote = config.FCRemote
	} else {
		done <- r.Name + ": failed...no suitable remote found"
	}
	// Choose the correct branch
	branches, _ := rw.ListBranches()
	targetName := remote + "/" + target
	if slices.Contains(branches, targetName) {
		rw.Checkout(target, remote)
		rw.Rebase(config.MasterBranch, remote)
		done <- targetName + " -> " + r.Name
	} else if rw.Branch == config.MasterBranch {
		// do nothing. Git pull maybe?
		rw.Rebase(config.MasterBranch, remote)
		done <- config.MasterBranch + " == " + r.Name
		return
	} else {
		rw.Checkout(config.MasterBranch, remote)
		done <- config.MasterBranch + " -> " + r.Name
	}

	done <- r.Name + ": done!"
}
