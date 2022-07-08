package workers

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strings"
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
	branch := init.CurrentBranch()
	remotes := init.Remotes()
	return &RepoWorker{init.RepoInfo, branch, remotes}
}

func (init *RepoWorkerInitializer) CurrentBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = init.RepoInfo.Path
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// Make a custom decoder for each of these
	branch, _ := rd.ReadString('\n')
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	branch = strings.Trim(branch, " \n\r")
	return branch
}

func (init *RepoWorkerInitializer) Remotes() []string {
	cmd := exec.Command("git", "remote")
	cmd.Dir = init.RepoInfo.Path
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	remotes := make([]string, 0)
	// Make a custom decoder for each of these
	r, err := rd.ReadString('\n')
	for err == nil {
		r = strings.Trim(r, " \n\r")
		remotes = append(remotes, r)
		r, err = rd.ReadString('\n')
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return remotes
}

func (w *RepoWorker) Update(remotes ...string) error {
	args := []string{"remote", "update"}
	args = append(args, remotes...)
	cmd := exec.Command("git", args...)
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

// TODO: Remove the log.Fatal here for better logging
func (w *RepoWorker) RevParse(object string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short=4", object)
	cmd.Dir = w.RepoInfo.Path
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	hash, _ := rd.ReadString('\n')
	hash = strings.Trim(hash, " \n\r")
	return hash, nil

}

//TODO: This will often return errors if there are no items to stash
func (w *RepoWorker) Stash() error {
	cmd := exec.Command("git", "stash", "--include-untracked")
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

// TODO: Remove the log.Fatal here for better logging
func (w *RepoWorker) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-a", "-l")
	cmd.Dir = w.RepoInfo.Path
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	branches := make([]string, 0)
	// Make a custom decoder for each of these
	b, err := rd.ReadString('\n')
	for err == nil {
		b = strings.Trim(b, " \n\r")
		branches = append(branches, b)
		b, err = rd.ReadString('\n')
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return branches, nil
}

func (w *RepoWorker) Rebase(targetBranch string, targetRemote string) error {
	targetRemoteBranch := targetRemote + "/" + targetBranch
	cmd := exec.Command("git", "rebase", targetRemoteBranch)
	cmd.Dir = w.RepoInfo.Path
	output, err := cmd.Output()
	if err != nil {
		msg := w.RepoInfo.Name + ": " + string(output)
		log.Println(msg) // TODO:  replace this with better logging, maybe a cli --debug flag?
		cmd2 := exec.Command("git", "rebase", "--abort")
		cmd2.Dir = cmd.Dir
		cmd2.Run()
		return err
	}
	return nil
}

func (w *RepoWorker) CheckoutRemote(targetBranch string, targetRemote string) error {
	targetRemoteBranch := targetRemote + "/" + targetBranch
	// The -B flag force creates a new branch if one already exists
	// The --track flag force
	cmd := exec.Command("git", "checkout", "-B", targetBranch, "--track", targetRemoteBranch)
	cmd.Dir = w.RepoInfo.Path
	output, err := cmd.Output()
	if err != nil {
		msg := w.RepoInfo.Name + ": " + string(output)
		log.Println(msg)
		return err
	}
	w.Branch = targetBranch
	return nil
}

func (w *RepoWorker) CheckoutLocal(targetBranch string, targetRemote string) error {
	cmd := exec.Command("git", "checkout", targetBranch)
	cmd.Dir = w.RepoInfo.Path
	output, err := cmd.Output()
	if err != nil {
		msg := w.RepoInfo.Name + ": " + string(output)
		log.Println(msg)
		return err
	}
	w.Branch = targetBranch
	return nil
}

func (w *RepoWorker) Print() {
	fmt.Printf("Repo: %s ; Branch: %s", w.RepoInfo.Name, w.Branch)
}
