// Package auth manages the Weeek API token stored in the environment or user config.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const tokenEnvironmentVariable = "WEEEK_TOKEN"

// ErrNoToken indicates that no Weeek API token is configured.
var ErrNoToken = errors.New("token not found: set WEEEK_TOKEN or run weeek auth login")

type config struct {
	Token string `json:"token"`
}

// Token returns the Weeek API token, preferring the environment over the config file.
func Token() (string, error) {
	if token := os.Getenv(tokenEnvironmentVariable); token != "" {
		return token, nil
	}

	path, err := configPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNoToken
	}
	if err != nil {
		return "", fmt.Errorf("reading token config: %w", err)
	}

	var stored config
	if err := json.Unmarshal(data, &stored); err != nil {
		return "", fmt.Errorf("decoding token config: %w", err)
	}
	if stored.Token == "" {
		return "", ErrNoToken
	}

	return stored.Token, nil
}

// Save stores a Weeek API token in the user config with owner-only permissions.
func Save(token string) (saveErr error) {
	if token == "" {
		return errors.New("token must not be empty")
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("creating token config directory: %w", err)
	}

	data, err := json.Marshal(config{Token: token})
	if err != nil {
		return fmt.Errorf("encoding token config: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".config.json-*")
	if err != nil {
		return fmt.Errorf("creating temporary token config: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if !removeTemporary {
			return
		}

		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			saveErr = errors.Join(
				saveErr,
				fmt.Errorf("removing temporary token config: %w", err),
			)
		}
	}()

	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf(
			"writing temporary token config: %w",
			errors.Join(err, temporary.Close()),
		)
	}
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf(
			"setting token config permissions: %w",
			errors.Join(err, temporary.Close()),
		)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("closing temporary token config: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("saving token config: %w", err)
	}

	removeTemporary = false
	return nil
}

// Delete removes the saved Weeek API token. It succeeds when no config exists.
func Delete() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting token config: %w", err)
	}

	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}

	return filepath.Join(home, ".config", "weeek", "config.json"), nil
}
