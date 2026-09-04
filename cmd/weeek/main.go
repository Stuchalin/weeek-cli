package main

import (
	"io"
	"os"

	"github.com/Stuchalin/weeek-cli/internal/api"
	"github.com/Stuchalin/weeek-cli/internal/auth"
	"github.com/Stuchalin/weeek-cli/internal/commands"
)

const apiBaseURL = "https://api.weeek.net/public/v1"

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	registry := commands.NewRegistry(version)
	ctx := &commands.Ctx{
		Stdout: stdout,
		Stderr: stderr,
	}
	if needsToken(args, registry) {
		token, err := auth.Token()
		if err != nil {
			commands.WriteError(stderr, err)
			return 1
		}
		ctx.Client = api.NewClient(apiBaseURL, token, nil)
	}

	return registry.Run(args, ctx)
}

func needsToken(args []string, registry commands.Registry) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "auth", "help", "version":
		return false
	}

	return registry.HasCommand(args)
}
