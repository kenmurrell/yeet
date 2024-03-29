package workers

import (
	"fmt"
)

type RepoWorker struct {
	RepoInfo *RepoInfo
	Branch   string
	Remotes  []string
}

type RepoWorkerInitializer struct {
	RepoInfo *RepoInfo
}

func (init *RepoWorkerInitializer) NewRepoWorker() (*RepoWorker, error) {
	branch, err := init.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("(%s): %s", init.RepoInfo.Name, err)
	}
	remotes, err := init.Remotes()
	if err != nil {
		return nil, fmt.Errorf("(%s): %s", init.RepoInfo.Name, err)
	}
	return &RepoWorker{init.RepoInfo, branch, remotes}, nil
}

func (init *RepoWorkerInitializer) CurrentBranch() (string, error) {
	args := []string{"branch", "--show-current"}
	cmd := GitCommand{args, init.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		if len(result.Output) == 1 {
			return result.Output[0], nil
		}
		return "DETACHED_HEAD", nil
	}
	return "", fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (init *RepoWorkerInitializer) Remotes() ([]string, error) {
	args := []string{"remote"}
	cmd := GitCommand{args, init.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return result.Output, nil
	}
	return nil, fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) Update(remotes ...string) error {
	args := []string{"remote", "update"}
	args = append(args, remotes...)
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return nil
	}
	return fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) RevParseObject(object string) (string, error) {
	args := []string{"rev-parse", "--short=4", object}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed && len(result.Output) == 1 {
		return result.Output[0], nil
	}
	return "", fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) RevParseUpstream(branch string) (string, error) {

	args := []string{"rev-parse", "--short=4", branch + "@{upstream}"}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed && len(result.Output) == 1 {
		return result.Output[0], nil
	}
	return "", fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) Stash() error {
	args := []string{"stash", "--include-untracked"}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return nil
	}
	return fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) BranchList() ([]string, error) {
	args := []string{"branch", "-a", "-l"}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return result.Output, nil
	}
	return nil, fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) StatusBranch() ([]string, error) {
	args := []string{"status", "-b", "--porcelain"}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return result.Output, nil
	}
	return nil, fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

func (w *RepoWorker) Rebase(targetBranch string) (bool, error) {
	args := []string{"rebase", targetBranch}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return true, nil
	} else if result.ErrorCode == 1 {
		abortArgs := []string{"rebase", "--abort"}
		abortCmd := GitCommand{abortArgs, w.RepoInfo.Path}
		abortResult := abortCmd.Run()
		if abortResult.Passed {
			return false, nil
		}
		return false, fmt.Errorf("%s failed with ErrorCode %d: %v", abortCmd.Print(), abortResult.ErrorCode, result.Output)
	}
	return false, fmt.Errorf("%s failed with ErrorCode %d: %v", cmd.Print(), result.ErrorCode, result.Output)
}

func (w *RepoWorker) CheckoutRemote(targetBranch string, targetRemote string) error {
	targetRemoteBranch := targetRemote + "/" + targetBranch
	// The -B flag force creates a new branch if one already exists
	// The --track flag force
	args := []string{"checkout", "-B", targetBranch, "--track", targetRemoteBranch}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if !result.Passed {
		return fmt.Errorf("%s failed with ErrorCode %d: %v", cmd.Print(), result.ErrorCode, result.Output)
	}
	w.Branch = targetBranch
	return nil
}

func (w *RepoWorker) CheckoutLocal(targetBranch string) error {
	args := []string{"checkout", targetBranch}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if !result.Passed {
		return fmt.Errorf("%s failed with ErrorCode %d: %v", cmd.Print(), result.ErrorCode, result.Output)
	}
	w.Branch = targetBranch
	return nil
}

func (w *RepoWorker) Print() {
	fmt.Printf("Repo: %s ; Branch: %s", w.RepoInfo.Name, w.Branch)
}
