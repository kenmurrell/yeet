package workers

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/go-color"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const (
	PASSED = iota
	FAILED = iota
	CNFLCT = iota
)

type ProgramConfig struct {
	MasterBranch string `yaml:"masterbranch"`
	FCRemote     string `yaml:"fcr"`
	RepoDir      string `yaml:"repodir"`
}

type WorkFlowResult struct {
	RepoName    string
	SuccessCode int
	Message     string
}

var config *ProgramConfig
var repolist *RepoList
var GOMAXPROCS int = 4

var RepolistFilename string = "repolist.json"

func loadconfig() *ProgramConfig {
	ex, _ := os.Executable()
	configPath := filepath.Join(filepath.Dir(ex), "config.yaml")
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln("The config.yaml file is missing!", err)
	}
	var tempconfig ProgramConfig
	err = yaml.Unmarshal(yamlFile, &tempconfig)
	if err != nil {
		panic(err)
	}
	if tempconfig.FCRemote == "" || tempconfig.RepoDir == "" || tempconfig.MasterBranch == "" {
		log.Fatalln("The config.yaml is missing values!")
	}
	return &tempconfig
}

func SetupCmd() {
	config = loadconfig()
	ex, _ := os.Executable()
	repoListPath := filepath.Join(filepath.Dir(ex), RepolistFilename)
	sw := SoloWorker{repoListPath, config.RepoDir}
	r, err := sw.GetList()
	if err != nil {
		msg := fmt.Sprintf("Error loading %s, you may need to run `yeet refresh` first?", RepolistFilename)
		log.Fatalln(msg)
	}
	repolist = r
}

func RefreshCmd() {
	config = loadconfig()
	ex, _ := os.Executable()
	repoListPath := filepath.Join(filepath.Dir(ex), RepolistFilename)
	sw := SoloWorker{repoListPath, config.RepoDir}
	err := sw.Refresh()
	if err != nil {
		panic(err)
	}
	r, _ := sw.GetList()
	n := len(r.RepoList)
	fmt.Printf("Loaded %d repositories into %s.", n, repoListPath)
}

func RebaseCmd(target string) {
	runtime.GOMAXPROCS(GOMAXPROCS)
	numCPUs := strconv.Itoa(GOMAXPROCS)
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
		fmt.Print(result.Format())
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
		done <- &WorkFlowResult{r.Name, FAILED, "Correct remote not found"}
		return
	}

	// Update info from the correct remote
	// Avoiding updating from all remotes here to save time
	err := rw.Update(remote)
	if err != nil {
		done <- &WorkFlowResult{r.Name, FAILED, "Error performing remote update: " + err.Error()}
		return
	}

	//if the repo is already on the target branch
	if rw.Branch == target {
		success, err := rw.Rebase(config.MasterBranch, remote)
		localHash, _ := rw.RevParse("HEAD")
		if err != nil {
			done <- &WorkFlowResult{r.Name, FAILED, "Error performing rebase: " + err.Error()}
		} else if !success {
			done <- &WorkFlowResult{r.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		}
		return
	}

	// Choose the correct branch
	branches, _ := rw.ListBranches()
	remoteTarget := fmt.Sprintf("remotes/%s/%s", remote, target)
	//if the repo is on another branch but has access to the target branch
	if slices.Contains(branches, remoteTarget) {
		branch := rw.Branch
		if branch == "" {
			branch = "DETACHED_HEAD"
		}
		err := rw.CheckoutRemote(target, remote)
		if err != nil {
			done <- &WorkFlowResult{r.Name, FAILED, "Error performing checkout: " + err.Error()}
		}
		success, err := rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- &WorkFlowResult{r.Name, FAILED, "Error performing rebase: " + err.Error()}
		} else if !success {
			done <- &WorkFlowResult{r.Name, CNFLCT, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		} else {
			done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		}
		return
	} else if slices.Contains(branches, target) {
		branch := rw.Branch
		if branch == "" {
			branch = "DETACHED_HEAD"
		}
		err := rw.CheckoutLocal(target)
		if err != nil {
			done <- &WorkFlowResult{r.Name, FAILED, "Error performing checkout: " + err.Error()}
		}
		done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		return
	}
	//if the repo is on the master branch and has no access to the target branch
	if rw.Branch == config.MasterBranch {
		// TODO: call these through goroutines at the same time
		remoteBranch := remote + "/" + config.MasterBranch
		localHash, lerr := rw.RevParse("HEAD")
		remoteHash, rerr := rw.RevParse(remoteBranch)
		if lerr != nil || rerr != nil {
			done <- &WorkFlowResult{r.Name, FAILED, "Error performing rev-parse: " + err.Error()}
			return
		}

		if localHash == remoteHash {
			done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			success, err := rw.Rebase(config.MasterBranch, remote)
			if err != nil {
				done <- &WorkFlowResult{r.Name, FAILED, "Error performing rebase: " + err.Error()}
			} else if !success {
				done <- &WorkFlowResult{r.Name, CNFLCT, fmt.Sprintf("[%s]: [%s] -> [%s]", rw.Branch, localHash, remoteHash)}
			} else {
				done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s]: [%s] -> [%s]", rw.Branch, localHash, remoteHash)}
			}
		}
		return
	}
	//if the repo is on another branch and has no access to the target branch
	branch := rw.Branch
	if branch == "" {
		branch = "DETACHED_HEAD"
	}
	err = rw.CheckoutRemote(config.MasterBranch, remote)
	if err != nil {
		done <- &WorkFlowResult{r.Name, FAILED, "Error performing checkout: " + err.Error()}
	}
	done <- &WorkFlowResult{r.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
}

func (r *WorkFlowResult) Format() string {
	var filler strings.Builder
	var fillerlen = 31 - len(r.Message)
	for i := 0; i < fillerlen; i++ {
		filler.WriteString(" ")
	}

	var status string
	switch r.SuccessCode {
	case PASSED:
		status = color.InGreen("PASSED")
	case FAILED:
		status = color.InRed("FAILED")
	case CNFLCT:
		status = color.InYellow("CNFLCT")
	}
	return fmt.Sprintf(" %s %s%s%s\n", status, r.Message, filler.String(), r.RepoName)
}
