package workers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

type SoloWorker struct {
	RepolistFilename string
	CodeDir          string
}

// Exported struct (via json)
type RepoInfo struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type RepoList struct {
	RepoList []*RepoInfo
}

func (worker *SoloWorker) Refresh() error {
	cmd := exec.Command("repo", "list", "-f")
	cmd.Dir = worker.CodeDir
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	list := make([]*RepoInfo, 0)
	line, err := rd.ReadString('\n')
	if err != nil {
		fmt.Printf("Error performing 'repo list', no entries found.\n")
		return err
	}
	for err == nil {
		chunks := strings.SplitN(line, " : ", 2)
		p := strings.Trim(chunks[0], " \n\r")
		n := strings.Trim(chunks[1], " \n\r")
		repoInfo := RepoInfo{Path: p, Name: n}
		list = append(list, &repoInfo)
		line, err = rd.ReadString('\n')
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	repoList := RepoList{list}
	jsontext, _ := json.MarshalIndent(&repoList, "", "\t")
	if err := ioutil.WriteFile(worker.RepolistFilename, jsontext, 0644); err != nil {
		return err
	}
	return nil
}

func (worker *SoloWorker) GetList() (*RepoList, error) {
	file, err := ioutil.ReadFile(worker.RepolistFilename)
	if err != nil {
		return nil, err
	}

	repolist := RepoList{}

	err = json.Unmarshal([]byte(file), &repolist)
	if err != nil {
		return nil, err
	}
	return &repolist, nil
}
