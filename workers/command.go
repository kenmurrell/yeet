package workers

import (
	"bufio"
	"os/exec"
	"strings"
)

type GitCommand struct {
	Args []string
	Path string
}

type GitCommandResult struct {
	Output    []string
	Passed    bool
	ErrorCode int
}

func (gcmd *GitCommand) Run() *GitCommandResult {
	cmd := exec.Command("git", gcmd.Args...)
	cmd.Dir = gcmd.Path
	stdout, err := cmd.StdoutPipe()
	rd := bufio.NewReader(stdout)
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		return &GitCommandResult{nil, false, 1}
	}
	lines := make([]string, 0)
	// Make a custom decoder for each of these
	b, err := rd.ReadString('\n')
	for err == nil {
		b = strings.Trim(b, " \n\r")
		lines = append(lines, b)
		b, err = rd.ReadString('\n')
	}
	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return &GitCommandResult{lines, false, exitError.ExitCode()}
		}
		return &GitCommandResult{lines, false, 1}
	}
	return &GitCommandResult{lines, true, 0}
}

func (gcmd *GitCommand) Print() string {
	var sb strings.Builder
	sb.WriteString("git ")
	for _, str := range gcmd.Args {
		sb.WriteString(str)
		sb.WriteString(" ")
	}
	return sb.String()
}
