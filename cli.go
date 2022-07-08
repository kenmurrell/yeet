package yeet

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
			Name:        "rebase",
			Usage:       "Rebase target branch onto the tip of main across all repos",
			Action:      entryPoint,
			Flags:       flags,
			UsageText:   "yeet rebase <targetbranch>",
			Description: "Rebases origin/<targetbranch> onto the tip of origin/main across all repos. All repositories that do not have the branch origin/<targetbranch> are updated to the tip of origin/main. repolist.json must exist.",
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
		refreshAction(cCtx)
	case "rebase":
		rebaseAction(cCtx)
	default:
		cli.ShowAppHelp(cCtx)
	}

	return nil
}

func refreshAction(cCtx *cli.Context) error {
	if cCtx.NArg() > 0 {
		fmt.Println("Refresh takes no arguments")
		return nil
	}

	workers.RefreshCmd()

	return nil
}

func rebaseAction(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		fmt.Printf("Rebase takes 1 argument, got %d\n", cCtx.NArg())
		return nil
	}

	branchName := cCtx.Args().Get(0)

	workers.SetupCmd()
	workers.RebaseCmd(branchName)

	return nil
}
