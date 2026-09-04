package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
	"github.com/Stuchalin/weeek-cli/internal/auth"
)

func registerAuthCommands(registry Registry) {
	registry.Register(authLoginCommand())
	registry.Register(authStatusCommand())
	registry.Register(authLogoutCommand())
}

func authLoginCommand() Command {
	return Command{
		Name:  "auth login",
		Usage: "auth login [--token TOKEN]",
		Short: "Save a Weeek API token",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
			token := fs.String("token", "", "Weeek API token")
			positionals, err := parseAll(fs, ctx.Args)
			if err != nil {
				return err
			}
			if len(positionals) != 0 {
				return &usageError{err: errors.New("auth login does not accept arguments")}
			}

			value := strings.TrimSpace(*token)
			if value == "" {
				value, err = promptToken(ctx.Stdin, ctx.Stderr)
				if err != nil {
					return &usageError{err: err}
				}
			}
			if err := auth.Save(value); err != nil {
				return err
			}

			path, err := auth.ConfigPath()
			if err != nil {
				return err
			}
			output := struct {
				OK     bool   `json:"ok"`
				Config string `json:"config"`
			}{
				OK:     true,
				Config: path,
			}
			if err := json.NewEncoder(ctx.Stdout).Encode(output); err != nil {
				return fmt.Errorf("writing auth login output: %w", err)
			}

			return nil
		},
	}
}

func authStatusCommand() Command {
	return Command{
		Name:  "auth status",
		Usage: "auth status",
		Short: "Check the configured Weeek API token",
		Run: func(ctx *Ctx) error {
			if len(ctx.Args) != 0 {
				return &usageError{err: errors.New("auth status does not accept arguments")}
			}

			token, err := auth.Token()
			if err != nil {
				return err
			}
			if ctx.NewAPIClient == nil {
				return errors.New("api client factory is not configured")
			}

			response, _, err := ctx.NewAPIClient(token).GetMe(context.Background())
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized {
					return &invalidTokenError{cause: apiErr}
				}

				return err
			}

			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func authLogoutCommand() Command {
	return Command{
		Name:  "auth logout",
		Usage: "auth logout",
		Short: "Remove the saved Weeek API token",
		Run: func(ctx *Ctx) error {
			if len(ctx.Args) != 0 {
				return &usageError{err: errors.New("auth logout does not accept arguments")}
			}
			if err := auth.Delete(); err != nil {
				return err
			}
			if err := json.NewEncoder(ctx.Stdout).Encode(struct {
				OK bool `json:"ok"`
			}{OK: true}); err != nil {
				return fmt.Errorf("writing auth logout output: %w", err)
			}

			return nil
		},
	}
}

type invalidTokenError struct {
	cause error
}

func (e *invalidTokenError) Error() string {
	return "token is invalid or expired"
}

func (e *invalidTokenError) Unwrap() error {
	return e.cause
}

func promptToken(reader io.Reader, writer io.Writer) (string, error) {
	if reader == nil {
		return "", errors.New("token is required")
	}
	if _, err := fmt.Fprint(writer, "Token: "); err != nil {
		return "", fmt.Errorf("writing token prompt: %w", err)
	}
	line, hidden, err := readTerminalLine(reader)
	if hidden {
		if _, newlineErr := fmt.Fprintln(writer); newlineErr != nil {
			err = errors.Join(err, fmt.Errorf("writing token prompt terminator: %w", newlineErr))
		}
		if err != nil {
			return "", err
		}
		token := strings.TrimSpace(line)
		if token == "" {
			return "", errors.New("token is required")
		}
		return token, nil
	}
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("reading token: %w", err)
		}

		return "", errors.New("token is required")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return "", errors.New("token is required")
	}

	return token, nil
}

func writeRawJSON(writer io.Writer, response json.RawMessage) error {
	if len(response) == 0 {
		response = json.RawMessage(`{}`)
	}
	if _, err := writer.Write(response); err != nil {
		return fmt.Errorf("writing api response: %w", err)
	}
	if bytes.HasSuffix(response, []byte("\n")) {
		return nil
	}
	if _, err := io.WriteString(writer, "\n"); err != nil {
		return fmt.Errorf("writing api response terminator: %w", err)
	}

	return nil
}
