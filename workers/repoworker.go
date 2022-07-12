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

func (init *RepoWorkerInitializer) NewRepoWorker() *RepoWorker {
	branch, err := init.CurrentBranch()
	if err != nil {
		panic(err)
	}
	remotes, err := init.Remotes()
	if err != nil {
		panic(err)
	}
	return &RepoWorker{init.RepoInfo, branch, remotes}
}

func (init *RepoWorkerInitializer) CurrentBranch() (string, error) {
	args := []string{"branch", "--show-current"}
	cmd := GitCommand{args, init.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		if len(result.Output) == 1 {
			return result.Output[0], nil
		}
		return "", fmt.Errorf("%s failed to find the current branch\n", cmd.Print())
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

func (w *RepoWorker) RevParse(object string) (string, error) {
	args := []string{"rev-parse", "--short=4", object}
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

func (w *RepoWorker) ListBranches() ([]string, error) {
	args := []string{"branch", "-a", "-l"}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return result.Output, nil
	}
	return nil, fmt.Errorf("%s failed with ErrorCode %d", cmd.Print(), result.ErrorCode)
}

// TODO: allow this to only take one argument
func (w *RepoWorker) Rebase(targetBranch string, targetRemote string) error {
	if targetRemote != "" {
		targetBranch = targetRemote + "/" + targetBranch
	}

	args := []string{"rebase", targetBranch}
	cmd := GitCommand{args, w.RepoInfo.Path}
	result := cmd.Run()
	if result.Passed {
		return nil
	} else if result.ErrorCode == 1 {
		fmt.Printf("%s rebase found merge conflicts, aborting...\n", w.RepoInfo.Name)
		abortArgs := []string{"rebase", "--abort"}
		abortCmd := GitCommand{abortArgs, w.RepoInfo.Path}
		abortResult := abortCmd.Run()
		if abortResult.Passed {
			return nil //TODO: add custom error type to return here
		}
		return fmt.Errorf("%s failed with ErrorCode %d: %v", abortCmd.Print(), abortResult.ErrorCode, result.Output)
	}
	return fmt.Errorf("%s failed with ErrorCode %d: %v", cmd.Print(), result.ErrorCode, result.Output)
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
