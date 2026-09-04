//go:build !darwin && !linux

package commands

import "io"

func readTerminalLine(io.Reader) (string, bool, error) {
	return "", false, nil
}
