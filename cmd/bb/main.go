package main

import (
	"os"

	"github.com/rbansal42/bitbucket-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
