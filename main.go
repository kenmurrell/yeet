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

	app := &cli.App{
		Name:   APP_NAME,
		Usage:  APP_USAGE,
		Action: entryPoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func entryPoint(cCtx *cli.Context) error {
	fmt.Printf("Cool functionality pending...")
	return nil
}
