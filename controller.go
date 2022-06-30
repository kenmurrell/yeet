package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/TwiN/go-color"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type ProgramConfig struct {
	MasterBranch string `yaml:"masterbranch"`
	FCRemote     string `yaml:"fcr"`
	RepoDir      string `yaml:"repodir"`
}

type WorkFlowResult struct {
	RepoName string
	Success  bool
	Message  string
}

var config *ProgramConfig
var repolist *RepoList
var maximumprocessors int = 4

var RepolistFilename string = "repolist.json"

func loadconfig() *ProgramConfig {
	var tempconfig ProgramConfig
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalln("The config.yaml file is missing!", err)
	}

	err = yaml.Unmarshal(yamlFile, &tempconfig)
	if err != nil {
		panic(err)
	}
	if tempconfig.FCRemote == "" || tempconfig.RepoDir == "" || tempconfig.MasterBranch == "" {
		log.Fatalln("The config.yaml is missing values!")
	}
	return &tempconfig
}

func setup() {
	config = loadconfig()
	sw := SoloWorker{RepolistFilename, config.RepoDir}
	r, err := sw.GetList()
	if err != nil {
		msg := fmt.Sprintf("Error loading %s, you may need to run `yeet refresh` first?", RepolistFilename)
		log.Fatalln(msg)
	}
	repolist = r
}

func refresh() {
	config = loadconfig()
	sw := SoloWorker{RepolistFilename, config.RepoDir}
	err := sw.Refresh()
	if err != nil {
		panic(err)
	}
	r, _ := sw.GetList()
	n := len(r.RepoList)
	fmt.Printf("Loaded %d repositories into %s.", n, RepolistFilename)
}

func run(target string) {
	runtime.GOMAXPROCS(maximumprocessors)
	numCPUs := strconv.Itoa(maximumprocessors)
	fmt.Printf("Checking out any %s branches using %s CPUs...\n", color.InYellow(target), color.InYellow(numCPUs))
	start := time.Now()
	done := make(chan *WorkFlowResult)
	defer close(done)
	var n int = len(repolist.RepoList)
	for _, r := range repolist.RepoList {
		go workflow(target, r, done)
	}
	fmt.Printf("Started %d processes...\n", n)

	for i := 0; i < n; i++ {
		result := <-done
		fmt.Printf("%s: ", result.RepoName)
		if result.Success {
			fmt.Printf(color.InGreen("PASSED"))
		} else {
			fmt.Printf(color.InRed("FAILED"))
		}
		fmt.Printf(" %s\n", result.Message)
	}

	elapsed := time.Since(start)
	fmt.Printf("Done, took %s", elapsed)
}

func workflow(target string, r *RepoInfo, done chan<- *WorkFlowResult) {
	init := RepoWorkerInitializer{r}
	rw := init.NewRepoWorker()
	// Stash current changes on branch
	_ = rw.Stash()
	// Select remote
	remotes := rw.Remotes
	var remote string
	if len(remotes) == 1 {
		remote = remotes[0]
	} else if slices.Contains(remotes, config.FCRemote) {
		remote = config.FCRemote
	} else {
		done <- &WorkFlowResult{r.Name, false, "Correct remote not found"}
		return
	}

	// Update info from the correct remote
	// Avoiding updating from all remotes here to save time
	err := rw.Update(remote)
	if err != nil {
		done <- &WorkFlowResult{r.Name, false, "Error performing remote update: " + err.Error()}
		return
	}

	// TODO: create a workflow for when target = masterbranch

	targetName := remote + "/" + target
	//if the repo is already on the target branch
	if rw.Branch == target {
		err := rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- &WorkFlowResult{r.Name, false, "Error performing rebase: " + err.Error()}
		} else {
			localHash, _ := rw.RevParse("HEAD")
			done <- &WorkFlowResult{r.Name, true, fmt.Sprintf("%s ", localHash)}
		}
		return
	}

	// Choose the correct branch
	branches, _ := rw.ListBranches()
	//if the repo is on another branch but has access to the target branch
	if slices.Contains(branches, targetName) {
		branch := rw.Branch
		if branch == "" {
			branch = "DETACHED_HEAD"
		}
		err := rw.Checkout(target, remote)
		if err != nil {
			done <- &WorkFlowResult{r.Name, false, "Error performing checkout: " + err.Error()}
		}
		err = rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- &WorkFlowResult{r.Name, false, "Error performing rebase: " + err.Error()}
			return
		}
		done <- &WorkFlowResult{r.Name, true, fmt.Sprintf("%s -> %s", branch, rw.Branch)}
		return
	}
	//if the repo is on the master branch and has no access to the target branch
	if rw.Branch == config.MasterBranch {
		// TODO: call these through goroutines at the same time
		remoteBranch := remote + "/" + config.MasterBranch
		localHash, lerr := rw.RevParse("HEAD")
		remoteHash, rerr := rw.RevParse(remoteBranch)
		if lerr != nil || rerr != nil {
			done <- &WorkFlowResult{r.Name, false, "Error performing rev-parse: " + err.Error()}
			return
		}

		if localHash == remoteHash {
			done <- &WorkFlowResult{r.Name, true, ""}
		} else {
			err := rw.Rebase(config.MasterBranch, remote)
			if err != nil {
				done <- &WorkFlowResult{r.Name, false, "Error performing rebase: " + err.Error()}
				return
			}
			done <- &WorkFlowResult{r.Name, true, fmt.Sprintf("%s -> %s", localHash, remoteHash)}
		}
		return
	}
	//if the repo is on another branch and has no access to the target branch
	branch := rw.Branch
	err = rw.Checkout(config.MasterBranch, remote)
	if err != nil {
		done <- &WorkFlowResult{r.Name, false, "Error performing checkout: " + err.Error()}
	}
	done <- &WorkFlowResult{r.Name, true, fmt.Sprintf("%s -> %s", branch, rw.Branch)}
}
