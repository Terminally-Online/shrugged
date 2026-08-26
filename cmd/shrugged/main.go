package main

import (
	"os"

	"git.ca.plug.to/terminally-online/shrugged/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
