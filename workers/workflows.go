package workers

import (
	"fmt"

	"golang.org/x/exp/slices"
)

type takeMessage struct {
	OldBranch string
	NewBranch string
	OldHEAD   string
	NewHEAD   string
	Error     string
	Addition  string
}

func (t *takeMessage) ToString() string {
	if t.Error == "" {
		return t.Error
	}
	str := fmt.Sprintf("[%s]", t.OldBranch)
	if t.NewBranch != "" {
		str += fmt.Sprintf("->[%s]", t.NewBranch)
	}
	str += fmt.Sprintf(": [%s]", t.OldHEAD)
	if t.NewHEAD != "" {
		str += fmt.Sprintf("->[%s]", t.NewHEAD)
	}
	if t.Addition != "" {
		str += fmt.Sprintf(" %s", t.Addition)
	}
	return str
}

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
	var remote string
	var localHEAD string
	var remoteHEAD string
	var branches []string
	var remoteTarget string
	var remoteMaster string
	rw, err := init.NewRepoWorker()
	if err != nil {
		panic(err)
	}
	branch := rw.Branch
	if branch == "" {
		branch = "D_HEAD"
	}
	message := takeMessage{OldBranch: branch}
	var status Status

	// Stash current changes on branch
	_ = rw.Stash()
	// Select remote
	if len(rw.Remotes) == 1 {
		remote = rw.Remotes[0]
	} else if slices.Contains(rw.Remotes, config.FCRemote) {
		remote = config.FCRemote
	} else {
		status = FAILED
		message.Error = "No suitable remote found."
		goto takeEND
	}
	remoteTarget = fmt.Sprintf("%s/%s", remote, config.MasterBranch)
	remoteMaster = fmt.Sprintf("%s/%s", remote, config.MasterBranch)

	// Update info from remote
	if err = rw.Update(remote); err != nil {
		status = FAILED
		message.Error = err.Error()
		goto takeEND
	}

	//CASE1: elif the target branch is the current branch
	if branch == target {
		localHEAD, _ = rw.RevParseObject("HEAD")
		message.OldHEAD = localHEAD
		remoteHEAD, err = rw.RevParseUpstream(target)
		if err != nil {
			status = CNFLCT
			message.Addition = "(no remote)"
			goto takeEND
		}
		if localHEAD == remoteHEAD {
			status = PASSED
		} else if rebaseSuccess, err := rw.Rebase(remoteTarget); err != nil {
			status = FAILED
			message.Error = err.Error()
			goto takeEND
		} else if !rebaseSuccess {
			status = CNFLCT
			goto takeEND
		} else {
			status = PASSED
			message.NewHEAD = remoteHEAD
		}
		// If currently on master, no need to rebase on master
		if branch == config.MasterBranch {
			goto takeEND
		}
		if rebaseSuccess, err := rw.Rebase(remoteMaster); err != nil {
			status = FAILED
			message.Error = err.Error()
		} else if !rebaseSuccess {
			status = CNFLCT
		} else {
			status = PASSED
			message.NewHEAD, _ = rw.RevParseObject("HEAD")
		}
		goto takeEND
	}

	//load branches
	branches, err = rw.BranchList()
	if err != nil {
		status = FAILED
		message.Error = err.Error()
		goto takeEND
	}

	//CASE2: elif the target branch exists locally
	if slices.Contains(branches, target) {
		if err := rw.CheckoutLocal(target); err != nil {
			status = FAILED
			message.Error = err.Error()
			goto takeEND
		}
		localHEAD, _ = rw.RevParseObject("HEAD")
		message.OldHEAD = localHEAD
		remoteHEAD, err = rw.RevParseUpstream(target)
		if err != nil {
			status = CNFLCT
			message.OldHEAD = localHEAD
			message.Addition = "(no remote)"
			goto takeEND
		}
		if localHEAD == remoteHEAD {
			status = PASSED
		} else if rebaseSuccess, err := rw.Rebase(remoteTarget); err != nil {
			status = FAILED
			message.Error = err.Error()
			goto takeEND
		} else if !rebaseSuccess {
			status = CNFLCT
			goto takeEND
		} else {
			status = PASSED
			message.NewHEAD = remoteHEAD
		}
		// If currently on master, no need to rebase on master
		if branch == config.MasterBranch {
			goto takeEND
		}
		if rebaseSuccess, err := rw.Rebase(remoteMaster); err != nil {
			status = FAILED
			message.Error = err.Error()
		} else if !rebaseSuccess {
			status = CNFLCT
		} else {
			status = PASSED
			message.NewHEAD, _ = rw.RevParseObject("HEAD")
		}
	} else if slices.Contains(branches, fmt.Sprintf("remotes/%s", remoteTarget)) {
		//CASE3: elif the target branch only exists remotely
		if err := rw.CheckoutRemote(target, remote); err != nil {
			status = FAILED
			message.Error = err.Error()
			goto takeEND
		}
		message.NewBranch = remoteTarget
		message.OldHEAD, _ = rw.RevParseObject("HEAD")
		if rebaseSuccess, err := rw.Rebase(remoteMaster); err != nil {
			status = FAILED
			message.Error = err.Error()
		} else if !rebaseSuccess {
			status = CNFLCT
		} else {
			status = PASSED
			message.NewHEAD, _ = rw.RevParseObject("HEAD")
		}
	} else {
		//CASE4: elif the repo has no access to the target branch
		if branch != config.MasterBranch {
			if err := rw.CheckoutLocal(config.MasterBranch); err != nil {
				status = FAILED
				message.Error = err.Error()
				goto takeEND
			}
			message.NewBranch = config.MasterBranch
		}
		localHEAD, _ = rw.RevParseObject("HEAD")
		message.OldHEAD = localHEAD
		remoteHEAD, err = rw.RevParseUpstream(config.MasterBranch)
		if err != nil {
			status = FAILED
			message.Error = err.Error()
			goto takeEND
		}
		if localHEAD == remoteHEAD {
			status = PASSED
		} else if rebaseSuccess, err := rw.Rebase(remoteMaster); err != nil {
			status = FAILED
			message.Error = err.Error()
		} else if !rebaseSuccess {
			status = CNFLCT
		} else {
			status = PASSED
			message.NewHEAD, _ = rw.RevParseObject("HEAD")
		}
	}

takeEND:
	done <- &WorkFlowResult{rw.RepoInfo.Name, status, message.ToString()}
	return
}
