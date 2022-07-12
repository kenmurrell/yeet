package workers_test

import (
	"os"
	"testing"
	workers "yeet/workers"
)

func TestRevParsePass(t *testing.T) {
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	args := []string{"rev-parse", "--short=6", "HEAD"}
	cmd := workers.GitCommand{args, currentPath}
	r := cmd.Run()
	if !r.Passed {
		t.Fatalf(`Command failed, exit code is %d`, r.ErrorCode)
	} else if len(r.Output) != 1 || len(r.Output[0]) != 6 {
		t.Fatalf(`Failed to collect hash`)
	}
}

func TestRevParseFail(t *testing.T) {
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	args := []string{"rev-parse", "0000"}
	cmd := workers.GitCommand{args, currentPath}
	r := cmd.Run()
	if r.Passed || r.ErrorCode != 128 {
		t.Fatalf(`Incorrect command incorrectly passed`)
	}
}
