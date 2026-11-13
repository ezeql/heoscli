package main

import (
	"os"

	"github.com/ezeql/heoscli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
