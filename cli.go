package main

import (
	"fmt"
	"log"
	workers "yeet/workers"

	"github.com/urfave/cli/v2"
)

// Globally available within the package. Set via the --debug,-d flag
var debugMode bool = false

type yCLI struct {
	args []string
}

func (ycli *yCLI) run() {
	const APP_NAME string = "yeet"
	const APP_USAGE string = "Rapidly switch between multi-repo branches"

	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Usage:       "Print debugging information",
			Destination: &debugMode,
		},
	}

	commands := []*cli.Command{
		{
			Name:        "refresh",
			Usage:       "Refresh repository list",
			Action:      entryPoint,
			Flags:       flags,
			UsageText:   "yeet refresh",
			Description: "Generates a list of repos across which to perform the target rebase and saves the results to repolist.json. Uses the `repo list` command.",
		},
		{
			Name:        "take",
			Usage:       "Checkout and rebase target branch onto the tip of main across all repos",
			Action:      entryPoint,
			Flags:       flags,
			UsageText:   "yeet take <targetbranch>",
			Description: "Rebases origin/<targetbranch> onto the tip of origin/main across all repos. All repositories that do not have the branch origin/<targetbranch> are updated to the tip of origin/main. repolist.json must exist.",
		},
		{
			Name:        "find",
			Usage:       "Searches all repositories for the chosen branch",
			Action:      entryPoint,
			Flags:       flags,
			UsageText:   "yeet find <targetbranch>",
			Description: "Searches all repos on their local and remotes for the target branch. Only print repos where something is found.",
		},
		{
			Name:        "status",
			Usage:       "Check the status of all repos",
			Action:      entryPoint,
			Flags:       flags,
			UsageText:   "yeet status",
			Description: "Checks the status of the current branch of every repo by checking the local and remote commit hashes.",
		},
	}

	app := &cli.App{
		Name:     APP_NAME,
		Usage:    APP_USAGE,
		Action:   entryPoint,
		Commands: commands,
	}

	if err := app.Run(ycli.args); err != nil {
		log.Fatal(err)
	}
}

func entryPoint(cCtx *cli.Context) error {
	if debugMode {
		fmt.Println("Running with debug ENABLED")
	}

	switch cCtx.Command.FullName() {
	case "refresh":
		return refreshAction(cCtx)
	case "take":
		return takeAction(cCtx)
	case "find":
		return findAction(cCtx)
	case "status":
		return statusAction(cCtx)
	default:
		cli.ShowAppHelp(cCtx)
	}

	return nil
}

func refreshAction(cCtx *cli.Context) error {
	if cCtx.NArg() > 0 {
		return fmt.Errorf("refresh takes no arguments")
	}
	workers.RefreshCmd()
	return nil
}

func takeAction(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		return fmt.Errorf("take needs 1 argument, got %d", cCtx.NArg())
	}
	branchName := cCtx.Args().Get(0)
	workers.SetupCmd()
	workers.TakeCmd(branchName)
	return nil
}

func findAction(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		return fmt.Errorf("take needs 1 argument, got %d", cCtx.NArg())
	}
	branchName := cCtx.Args().Get(0)
	workers.SetupCmd()
	workers.FindCmd(branchName)
	return nil
}

func statusAction(cCtx *cli.Context) error {
	if cCtx.NArg() > 0 {
		return fmt.Errorf("take needs no arguments")
	}
	workers.SetupCmd()
	workers.StatusCmd()
	return nil
}
