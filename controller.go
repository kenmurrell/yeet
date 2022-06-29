package main

import (
	"fmt"
	"io/ioutil"
	"time"

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
	start := time.Now()
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

	elapsed := time.Since(start)
	fmt.Printf("Done, took %s", elapsed)
}

func workflow(target string, r *RepoInfo, done chan<- string) {
	init := RepoWorkerInitializer{r}
	rw := init.NewRepoWorker()
	// Stash current changes on branch
	rw.Stash()
	// Select remote
	remotes := rw.Remotes
	var remote string
	if len(remotes) == 1 {
		remote = remotes[0]
	} else if slices.Contains(remotes, config.FCRemote) {
		remote = config.FCRemote
	} else {
		done <- r.Name + ": failed...no suitable remote found"
	}

	// Update info from the correct remote
	// Avoiding updating from all remotes here to save time
	rw.Update(remote)
	// Choose the correct remote

	// Choose the correct branch
	branches, _ := rw.ListBranches()
	targetName := remote + "/" + target
	if rw.Branch == target {
		// TODO: call these through goroutines at the same time
		localHash, _ := rw.RevParse("HEAD")
		remoteHash, _ := rw.RevParse(targetName)
		if localHash != remoteHash {
			err := rw.Rebase(config.MasterBranch, remote)
			if err != nil {
				done <- r.Name + ": FAILED"
			}
			done <- r.Name + ": " + rw.Branch + " -> " + remoteHash
		} else {
			done <- r.Name + ": " + rw.Branch
		}
	} else if slices.Contains(branches, targetName) {
		branch := rw.Branch
		err := rw.Checkout(target, remote)
		if err != nil {
			done <- r.Name + ": FAILED"
		}
		err = rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- r.Name + ": FAILED"
		}
		done <- r.Name + ": " + branch + " -> " + rw.Branch
	} else if rw.Branch == config.MasterBranch {
		// TODO: call these through goroutines at the same time
		localHash, _ := rw.RevParse("HEAD")
		remoteHash, _ := rw.RevParse(targetName)
		if localHash != remoteHash {
			err := rw.Rebase(config.MasterBranch, remote)
			if err != nil {
				done <- r.Name + ": FAILED"
			}
			done <- r.Name + ": " + rw.Branch + " -> " + remoteHash
		} else {
			done <- ""
		}
	} else {
		branch := rw.Branch
		_ = rw.Checkout(config.MasterBranch, remote)
		done <- r.Name + ": " + branch + " -> " + rw.Branch
	}
}
