package main

import (
	"io"
	"os"

	"github.com/Stuchalin/weeek-cli/internal/api"
	"github.com/Stuchalin/weeek-cli/internal/auth"
	"github.com/Stuchalin/weeek-cli/internal/commands"
)

var apiBaseURL = "https://api.weeek.net/public/v1"

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithInput(args, os.Stdin, stdout, stderr)
}

func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	registry := commands.NewRegistry(version)
	ctx := &commands.Ctx{
		ResolveToken: auth.Token,
		NewAPIClient: func(token string) *api.Client {
			return api.NewClient(apiBaseURL, token, nil)
		},
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	return registry.Run(args, ctx)
}
