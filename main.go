package main

import "os"

func main() {
	cli := yCLI{os.Args}
	cli.run()
}
