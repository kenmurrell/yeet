package main

import (
	"os/exec"
	"log"
	"bufio"
	"strings"
	"encoding/json"
	"io/ioutil"
)

type SoloWorker struct {
	repolist_filename string
	codedir string
}

type Repo struct {
	Path string
	Name string 
}

type RepoList struct {
	RepoList []*Repo
}

func (worker *SoloWorker) Referesh() {
	cmd := exec.Command("repo", "list", "-f")
	cmd.Dir = worker.codedir
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var repolist RepoList
	line, err := rd.ReadString('\n')
	for err == nil {
		chunks := strings.SplitN(line, " : ", 2)
		p := strings.Trim(chunks[0], " \n\r")
		n := strings.Trim(chunks[1], " \n\r")
		repo := Repo{Path: p, Name: n}
		repolist.RepoList = append(repolist.RepoList, &repo)
		line, err = rd.ReadString('\n')
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	jsontext, err := json.MarshalIndent(&repolist, "", "\t")
	_ = ioutil.WriteFile(worker.repolist_filename, jsontext, 0644)
}

func (worker *SoloWorker) GetList() *RepoList {
	file, _ := ioutil.ReadFile(worker.repolist_filename)
 
	repolist := RepoList{}
 
	_ = json.Unmarshal([]byte(file), &repolist)
	return &repolist
}