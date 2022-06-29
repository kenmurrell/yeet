package main

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

func (w *RepoWorker) Update() error {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

func (w *RepoWorker) Stash() error {
	cmd := exec.Command("git", "stash", "--include-untracked")
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

func (w *RepoWorker) Unstash() error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

func (w *RepoWorker) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "-l")
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
	err := cmd.Run()
	return err
}

func (w *RepoWorker) Checkout(targetBranch string, targetRemote string) error {
	targetRemoteBranch := targetRemote + "/" + targetBranch
	// The -B flag force creates a new branch if one already exists
	// The --track flag force
	cmd := exec.Command("git", "checkout", "-B", targetBranch, "--track", targetRemoteBranch)
	cmd.Dir = w.RepoInfo.Path
	err := cmd.Run()
	return err
}

func (w *RepoWorker) Print() {
	fmt.Printf("Repo: %s ; Branch: %s", w.RepoInfo.Name, w.Branch)
}
