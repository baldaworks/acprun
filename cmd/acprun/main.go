package main

import (
	"os"

	"github.com/baldaworks/acprun/internal/cli"
)

func main() {
	exitCode := cli.Execute()
	os.Exit(exitCode)
}
