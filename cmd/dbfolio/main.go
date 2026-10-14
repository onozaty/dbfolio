package main

import (
	"os"

	"github.com/onozaty/dbfolio/internal/cli"
)

// version is set at release build time via -ldflags "-X main.version=...".
var version string

func main() {
	if err := cli.NewRootCmd(version).Execute(); err != nil {
		// cobra has already printed the error.
		os.Exit(1)
	}
}
