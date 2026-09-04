//go:build darwin || linux

package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
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

	var restoreErr error
	var restoreOnce sync.Once
	restore := func() error {
		restoreOnce.Do(func() {
			restoreErr = setTerminalState(file.Fd(), original)
		})
		return restoreErr
	}

	signals := make(chan os.Signal, 1)
	signalDone := make(chan struct{})
	signalWatcherDone := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer close(signalWatcherDone)
		restoreTerminalOnSignal(signals, signalDone, restore, func() {
			signal.Stop(signals)
		}, forwardTerminalSignal)
	}()

	if err := setTerminalState(file.Fd(), &hiddenState); err != nil {
		signal.Stop(signals)
		close(signalDone)
		<-signalWatcherDone
		return "", true, fmt.Errorf("disabling terminal echo: %w", err)
	}
	defer func() {
		restoreErr := restore()
		signal.Stop(signals)
		close(signalDone)
		<-signalWatcherDone
		if restoreErr != nil {
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

func restoreTerminalOnSignal(
	signals <-chan os.Signal,
	done <-chan struct{},
	restore func() error,
	stop func(),
	forward func(os.Signal),
) {
	select {
	case received := <-signals:
		_ = restore()
		stop()
		forward(received)
	case <-done:
		select {
		case received := <-signals:
			_ = restore()
			forward(received)
		default:
		}
	}
}

func forwardTerminalSignal(received os.Signal) {
	process, err := os.FindProcess(os.Getpid())
	if err == nil {
		err = process.Signal(received)
	}
	if err != nil {
		os.Exit(1)
	}
}
