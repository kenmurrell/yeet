package workers

import (
	"fmt"

	"golang.org/x/exp/slices"
)

func statusWorkflow(init *RepoWorkerInitializer, done chan<- *WorkFlowResult) {
	var wfr *WorkFlowResult
	var localHEAD string
	var remoteHEAD string
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
		wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, "Error performing remote update: " + err.Error()}
		goto statusEND
	}
	localHEAD, _ = rw.RevParseObject("HEAD")
	remoteHEAD, err = rw.RevParseUpstream(rw.Branch)
	if err != nil {
		wfr = &WorkFlowResult{rw.RepoInfo.Name, CURRNT, fmt.Sprintf("[%s]: [%s] (no upstream)", rw.Branch, localHEAD)}
	} else if localHEAD == remoteHEAD {
		wfr = &WorkFlowResult{rw.RepoInfo.Name, CURRNT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHEAD)}
	} else {
		wfr = &WorkFlowResult{rw.RepoInfo.Name, BEHIND, fmt.Sprintf("[%s]: [%s]<->[%s]", rw.Branch, localHEAD, remoteHEAD)}
	}
statusEND:
	done <- wfr
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
	var message string
	var localHEAD string
	var remoteHEAD string
	var branches []string
	remoteTarget := fmt.Sprintf("%s/%s", remote, config.MasterBranch)
	fmtRemoteTarget := fmt.Sprintf("remotes/%s/%s", remote, target)
	remoteMasterBranch := fmt.Sprintf("%s/%s", remote, config.MasterBranch)

	// Update info from remote
	if err = rw.Update(remote); err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error performing remote update: %s", err.Error())}
		goto takeEND
	}

	//CASE1: elif the target branch is the current branch
	if rw.Branch == target {
		localHEAD, _ = rw.RevParseObject("HEAD")
		remoteHEAD, err = rw.RevParseUpstream(target)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s] (no remote)", rw.Branch, localHEAD)}
			goto exitcase1
		}
		if localHEAD == remoteHEAD {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHEAD)}
		} else if rebaseSuccess, err := rw.Rebase(remoteTarget); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
			goto exitcase1
		} else if !rebaseSuccess {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("[%s]: [%s]", rw.Branch, localHEAD)}
			goto exitcase1
		} else {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("[%s]: [%s]->[%s]", rw.Branch, localHEAD, remoteHEAD)}
		}
		// If currently on master, no need to rebase on master
		if rw.Branch == config.MasterBranch {
			goto exitcase1
		}
		if rebaseSuccess, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
		} else if !rebaseSuccess {
			wfr.Status = CNFLCT
		} else {
			newLocalHEAD, _ := rw.RevParseObject("HEAD")
			wfr.Message = fmt.Sprintf("[%s]: [%s]->[%s]", rw.Branch, localHEAD, newLocalHEAD)
		}
	exitcase1:
		done <- wfr
		return
	}

	//load branches
	branches, err = rw.BranchList()
	if err != nil {
		done <- &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error getting branch names: %s", err.Error())}
		goto takeEND
	}

	//CASE2: elif the target branch exists locally
	if slices.Contains(branches, target) {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if err := rw.CheckoutLocal(target); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error checking out %s: %s", target, err.Error())}
			goto exitcase2
		}
		message = fmt.Sprintf("[%s]->[%s]", prevBranch, target)
		localHEAD, _ = rw.RevParseObject("HEAD")
		remoteHEAD, err = rw.RevParseUpstream(target)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("%s: [%s] (no remote)", message, localHEAD)}
			goto exitcase2
		}
		if localHEAD == remoteHEAD {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]", message, localHEAD)}
		} else if rebaseSuccess, err := rw.Rebase(remoteTarget); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
			goto exitcase2
		} else if !rebaseSuccess {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("%s: [%s]", message, localHEAD)}
			goto exitcase2
		} else {
			newLocalHEAD, _ := rw.RevParseObject("HEAD")
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]->[%s]", message, localHEAD, newLocalHEAD)}
		}
		// If currently on master, no need to rebase on master
		if rw.Branch == config.MasterBranch {
			goto exitcase2
		}
		if rebaseSuccess, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
		} else if !rebaseSuccess {
			wfr.Status = CNFLCT
		} else {
			newLocalHEAD, _ := rw.RevParseObject("HEAD")
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]->[%s]", message, localHEAD, newLocalHEAD)}
		}
	exitcase2:
		done <- wfr
		return
	}

	//CASE3: elif the target branch only exists remotely
	if slices.Contains(branches, fmtRemoteTarget) {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if err := rw.CheckoutRemote(target, remote); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Cannot checkout remote branch %s: %s", target, err.Error())}
			goto exitcase3
		}
		message = fmt.Sprintf("[%s]->[%s]", prevBranch, target)
		localHEAD, _ = rw.RevParseObject("HEAD")
		if rebaseSuccess, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
		} else if !rebaseSuccess {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("%s: [%s]", message, localHEAD)}
		} else {
			newLocalHEAD, _ := rw.RevParseObject("HEAD")
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]->[%s]", message, localHEAD, newLocalHEAD)}
		}
	exitcase3:
		done <- wfr
		return
	}

	//CASE4: elif the repo has no access to the target branch
	if true {
		prevBranch := rw.Branch
		if prevBranch == "" {
			prevBranch = "DETACHED_HEAD"
		}
		if rw.Branch == config.MasterBranch {
			message = fmt.Sprintf("[%s]", config.MasterBranch)
		} else {
			if err := rw.CheckoutLocal(config.MasterBranch); err != nil {
				wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, fmt.Sprintf("Error checking out %s: %s", target, err.Error())}
				goto exitcase4
			}
			message = fmt.Sprintf("[%s]->[%s]", prevBranch, config.MasterBranch)
		}

		localHEAD, _ = rw.RevParseObject("HEAD")
		remoteHEAD, err = rw.RevParseUpstream(config.MasterBranch)
		if err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("Cannot update to remote: %s", err.Error())}
			goto exitcase4
		}
		if localHEAD == remoteHEAD {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]", message, localHEAD)}
		} else if rebaseSuccess, err := rw.Rebase(remoteMasterBranch); err != nil {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, FAILED, err.Error()}
		} else if !rebaseSuccess {
			wfr = &WorkFlowResult{rw.RepoInfo.Name, CNFLCT, fmt.Sprintf("%s: [%s]", message, localHEAD)}
		} else {
			newLocalHEAD, _ := rw.RevParseObject("HEAD")
			wfr = &WorkFlowResult{rw.RepoInfo.Name, PASSED, fmt.Sprintf("%s: [%s]->[%s]", message, localHEAD, newLocalHEAD)}
		}
	exitcase4:
		done <- wfr
		return
	}

takeEND:
	return
}
