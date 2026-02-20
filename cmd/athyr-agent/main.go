package main

import (
	"os"

	"github.com/athyr-tech/athyr-agent/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
