package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	const APP_NAME string = "yeet"
	const APP_USAGE string = "Rapidly switch between multi-repo branches"

	commands := []*cli.Command{
		{
			Name:   "refresh",
			Usage:  "Refresh repository list",
			Action: refreshAction,
		},
		{
			Name:   "rebase",
			Usage:  "Rebase target branch onto the tip of main on all repos",
			Action: rebaseAction,
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
	fmt.Println("Cool functionality pending...")
	return nil
}

func refreshAction(cCtx *cli.Context) error {
	if cCtx.NArg() > 0 {
		fmt.Println("Refresh takes no arguments")
		return nil
	}

	fmt.Println("TODO: Perform refresh here")

	return nil
}

func rebaseAction(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		fmt.Printf("Rebase takes 1 argument, got %d\n", cCtx.NArg())
		return nil
	}

	branchName := cCtx.Args().Get(0)

	fmt.Printf("TODO: Perform rebase here with branch %s\n", branchName)

	return nil
}
