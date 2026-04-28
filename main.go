package main

import (
	"os"

	"github.com/larasense/larasense-limbo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
