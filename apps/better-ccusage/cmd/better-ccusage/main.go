package main

import (
	"fmt"
	"os"

	"github.com/cobra91/better-ccusage/pkg/terminal"
)

const version = "0.0.0-dev"

func main() {
	log := terminal.NewLoggerFromEnv()
	root := newRootCmd(log)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
