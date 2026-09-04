//go:build darwin || linux

package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"syscall"
)

func readTerminalLine(reader io.Reader) (line string, hidden bool, err error) {
	file, ok := reader.(interface{ Fd() uintptr })
	if !ok {
		return "", false, nil
	}

	original, err := getTerminalState(file.Fd())
	if errors.Is(err, syscall.ENOTTY) {
		return "", false, nil
	}
	if err != nil {
		return "", true, fmt.Errorf("reading terminal settings: %w", err)
	}

	hiddenState := *original
	hiddenState.Lflag &^= syscall.ECHO
	if err := setTerminalState(file.Fd(), &hiddenState); err != nil {
		return "", true, fmt.Errorf("disabling terminal echo: %w", err)
	}
	defer func() {
		if restoreErr := setTerminalState(file.Fd(), original); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restoring terminal echo: %w", restoreErr))
		}
	}()

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return "", true, fmt.Errorf("reading token: %w", scanErr)
		}
		return "", true, nil
	}
	return scanner.Text(), true, nil
}
