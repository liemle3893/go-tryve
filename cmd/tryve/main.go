package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "tryve %s\n", version)
	os.Exit(0)
}
