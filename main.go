package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// Globally available within the package. Set via the --debug,-d flag
var debugMode bool = false

func main() {
	const APP_NAME string = "yeet"
	const APP_USAGE string = "Rapidly switch between multi-repo branches"

	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Usage:       "Set to print debugging information",
			Destination: &debugMode,
		},
	}

	commands := []*cli.Command{
		{
			Name:   "refresh",
			Usage:  "Refresh repository list",
			Action: entryPoint,
			Flags:  flags,
		},
		{
			Name:   "rebase",
			Usage:  "Rebase target branch onto the tip of main on all repos",
			Action: entryPoint,
			Flags:  flags,
		},
	}

	app := &cli.App{
		Name:     APP_NAME,
		Usage:    APP_USAGE,
		Action:   entryPoint,
		Commands: commands,
	}

	if err := app.Run(os.Args); err != nil {
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

	refresh()

	return nil
}

func rebaseAction(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		fmt.Printf("Rebase takes 1 argument, got %d\n", cCtx.NArg())
		return nil
	}

	branchName := cCtx.Args().Get(0)

	setup()
	run(branchName)

	return nil
}
