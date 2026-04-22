package main

import (
	"fmt"
	"os"

	"github.com/liemle3893/autoflow/internal/cli"
)

var version = "dev"

func main() {
	cli.SetSandboxHostVersion(version)
	root := cli.NewRoot(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
