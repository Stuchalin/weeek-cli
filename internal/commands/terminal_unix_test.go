//go:build darwin || linux

package commands

import (
	"os"
	"syscall"
	"testing"
)

func TestRestoreTerminalOnSignal(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	events := make([]string, 0, 3)
	received := os.Signal(nil)

	signals <- syscall.SIGINT
	restoreTerminalOnSignal(
		signals,
		done,
		func() error {
			events = append(events, "restore")
			return nil
		},
		func() {
			events = append(events, "stop")
		},
		func(receivedSignal os.Signal) {
			events = append(events, "forward")
			received = receivedSignal
		},
	)

	wantEvents := []string{"restore", "stop", "forward"}
	if len(events) != len(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
	for index, want := range wantEvents {
		if events[index] != want {
			t.Fatalf("event %d = %q, want %q; all events = %v", index, events[index], want, events)
		}
	}
	if received != syscall.SIGINT {
		t.Errorf("forwarded signal = %v, want %v", received, syscall.SIGINT)
	}
}

func TestRestoreTerminalForwardsSignalQueuedDuringShutdown(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	restored := false
	received := os.Signal(nil)

	signals <- syscall.SIGTERM
	close(done)
	restoreTerminalOnSignal(
		signals,
		done,
		func() error {
			restored = true
			return nil
		},
		func() {},
		func(receivedSignal os.Signal) {
			received = receivedSignal
		},
	)

	if !restored {
		t.Error("terminal was not restored before forwarding queued signal")
	}
	if received != syscall.SIGTERM {
		t.Errorf("forwarded signal = %v, want %v", received, syscall.SIGTERM)
	}
}
