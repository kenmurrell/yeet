package workers

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"
)

func statusWorkflow(init *RepoWorkerInitializer, done chan<- *WorkFlowResult) {
	rw := init.NewRepoWorker()
	remotes := rw.Remotes
	var remote string
	if len(remotes) == 1 {
		remote = remotes[0]
	} else if slices.Contains(remotes, config.FCRemote) {
		remote = config.FCRemote
	} else {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Correct remote not found"}
		return
	}
	err := rw.Update(remote)
	if err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing remote update: " + err.Error()}
		return
	}

	// TODO: there's a better way of doing this...
	var exp = regexp.MustCompile(`(?P<prefix>\#\#\s)(?P<local>[a-zA-Z0-9\_\-\/]+)(?P<split>\.\.\.)?(?P<remote>[a-zA-Z0-9\-\_\/]+)?`)
	output, err := rw.StatusBranch()
	if err != nil {
		panic(err)
	}
	match := exp.FindStringSubmatch(output[0])
	result := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 {
			result[name] = match[i]
		}
	}
	if result["local"] != rw.Branch {
		panic("This isnt supposed to happen...")
	}

	if result["remote"] == "" {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("[%s]: No upstream", rw.Branch)}
		return
	}

	localHash, _ := rw.RevParse("HEAD")
	remoteHash, err := rw.RevParse(result["remote"])
	if err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing rev-parse: " + err.Error()}
		return
	}
	if localHash == remoteHash {
		done <- &WorkFlowResult{rw.RepoInfo.Name, CURRNT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
	} else {
		done <- &WorkFlowResult{rw.RepoInfo.Name, BEHIND, fmt.Sprintf("[%s]: [%s]<->[%s]", rw.Branch, localHash, remoteHash)}
	}
}

func takeWorkflow(target string, init *RepoWorkerInitializer, done chan<- *WorkFlowResult) {
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
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Correct remote not found"}
		return
	}

	// Update info from the correct remote
	// Avoiding updating from all remotes here to save time
	err := rw.Update(remote)
	if err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing remote update: " + err.Error()}
		return
	}

	//if the repo is already on the target branch
	if rw.Branch == target {
		localHash, _ := rw.RevParse("HEAD")
		success, err := rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing rebase: " + err.Error()}
		} else if !success {
			done <- &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
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
			done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing checkout: " + err.Error()}
		}
		success, err := rw.Rebase(config.MasterBranch, remote)
		if err != nil {
			done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing rebase: " + err.Error()}
		} else if !success {
			done <- &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		} else {
			done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		}
		return
	} else if slices.Contains(branches, target) {
		branch := rw.Branch
		if branch == "" {
			branch = "DETACHED_HEAD"
		}
		err := rw.CheckoutLocal(target)
		if err != nil {
			done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing checkout: " + err.Error()}
		}
		done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
		return
	}
	//if the repo is on the master branch and has no access to the target branch
	if rw.Branch == config.MasterBranch {
		// TODO: call these through goroutines at the same time
		remoteBranch := remote + "/" + config.MasterBranch
		localHash, _ := rw.RevParse("HEAD")
		remoteHash, err := rw.RevParse(remoteBranch)
		if err != nil {
			done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing rev-parse: " + err.Error()}
			return
		}

		if localHash == remoteHash {
			done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			success, err := rw.Rebase(config.MasterBranch, remote)
			if err != nil {
				done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing rebase: " + err.Error()}
			} else if !success {
				done <- &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s] -> [%s]", rw.Branch, localHash, remoteHash)}
			} else {
				done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s] -> [%s]", rw.Branch, localHash, remoteHash)}
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
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing checkout: " + err.Error()}
	}
	done <- &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", branch, rw.Branch)}
}
