package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		if err := json.NewEncoder(stdout).Encode(map[string]string{"version": version}); err != nil {
			fmt.Fprintf(stderr, "failed to write version: %v\n", err)
			return 1
		}

		return 0
	}

	fmt.Fprintln(stderr, "usage: weeek version")
	return 2
}
