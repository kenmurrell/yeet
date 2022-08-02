package workers

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"
)

func statusWorkflow(init *RepoWorkerInitializer, done chan<- *WorkFlowResult) {
	rw, err := init.NewRepoWorker()
	if err != nil {
		panic(err)
	}
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
	err = rw.Update(remote)
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

	localHash, _ := rw.RevParseObject("HEAD")
	remoteHash, err := rw.RevParseObject(result["remote"])
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
	rw, err := init.NewRepoWorker()
	if err != nil {
		panic(err)
	}
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
	var wfr *WorkFlowResult
	var localHash string
	var remoteHash string
	var branches []string
	remoteTarget := fmt.Sprintf("%s/%s", remote, config.MasterBranch)
	fmtRemoteTarget := fmt.Sprintf("remotes/%s/%s", remote, target)
	remoteMasterBranch := fmt.Sprintf("%s/%s", remote, config.MasterBranch)

	// Update info from remote
	if err = rw.Update(remote); err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing remote update: %s", err.Error())}
		goto END
	}

	//CASE1: if the target branch is the master branch
	if config.MasterBranch == target {
		if err = rw.CheckoutLocal(config.MasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error checking out %s: %s", config.MasterBranch, err.Error())}
			goto exitcase1
		}
		localHash, _ = rw.RevParseObject("HEAD")
		remoteHash, err = rw.RevParseUpstream(target)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rev-parse: %s", err.Error())}
			goto exitcase1
		}
		if localHash == remoteHash {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else if selfrebase, err := rw.Rebase(remoteTarget); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !selfrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		}
	exitcase1:
		done <- wfr
		return
	}

	//CASE2: elif the target branch is the current branch
	if rw.Branch == target {
		localHash, _ = rw.RevParseObject("HEAD")
		remoteHash, err = rw.RevParseUpstream(target)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("Cannot update to remote: %s", err.Error())}
			goto exitcase2
		}
		if localHash == remoteHash {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else if selfrebase, err := rw.Rebase(remoteTarget); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !selfrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else if masterrebase, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !masterrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		}
	exitcase2:
		done <- wfr
		return
	}

	//load branches
	branches, err = rw.BranchList()
	if err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error getting branch names: %s", err.Error())}
		goto END
	}

	//CASE3: elif the target branch exists locally
	if slices.Contains(branches, target) {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if err := rw.CheckoutLocal(target); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error checking out %s: %s", target, err.Error())}
			goto exitcase3
		}
		localHash, _ = rw.RevParseObject("HEAD")
		remoteHash, err = rw.RevParseUpstream(target)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("Cannot update to remote: %s", err.Error())}
			goto exitcase3
		}
		if localHash == remoteHash {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", prevBranch, target)}
		} else if selfrebase, err := rw.Rebase(remoteTarget); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !selfrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s] -> [%s]", prevBranch, target)}
		} else if masterrebase, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !masterrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s] -> [%s]", prevBranch, target)}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", prevBranch, target)}
		}
	exitcase3:
		done <- wfr
		return
	}

	//CASE4: elif the target branch exists remotely
	if slices.Contains(branches, fmtRemoteTarget) {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if err := rw.CheckoutRemote(target, remote); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Cannot checkout remote branch %s: %s", target, err.Error())}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", prevBranch, target)}
		}
		done <- wfr
		return
	}

	//CASE5: elif the repo is not on the master branch
	if rw.Branch != config.MasterBranch {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if err := rw.CheckoutLocal(config.MasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error checking out %s: %s", target, err.Error())}
			goto exitcase5
		}
		localHash, _ = rw.RevParseObject("HEAD")
		remoteHash, err = rw.RevParseUpstream(config.MasterBranch)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("Cannot update to remote: %s", err.Error())}
			goto exitcase5
		}
		if localHash == remoteHash {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", prevBranch, config.MasterBranch)}
		} else if selfrebase, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !selfrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s] -> [%s]", prevBranch, config.MasterBranch)}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s] -> [%s]", prevBranch, config.MasterBranch)}
		}
	exitcase5:
		done <- wfr
		return
	}

	//CASE6: elif the repo has no access to target and is on master
	if true {
		localHash, _ = rw.RevParseObject("HEAD")
		remoteHash, err = rw.RevParseUpstream(config.MasterBranch)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("Cannot update to remote: %s", err.Error())}
			goto exitcase6
		}
		if localHash == remoteHash {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else if selfrebase, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing rebase: %s", err.Error())}
		} else if !selfrebase {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHash)}
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s] -> [%s]", rw.Branch, localHash, remoteHash)}
		}
	exitcase6:
		done <- wfr
		return
	}

END:
	return
}
