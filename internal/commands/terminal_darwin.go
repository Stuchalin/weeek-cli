//go:build darwin

package commands

import (
	"syscall"
	"unsafe"
)

func getTerminalState(fd uintptr) (*syscall.Termios, error) {
	state := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGETA),
		uintptr(unsafe.Pointer(state)),
		0,
		0,
		0,
	)
	if errno != 0 {
		return nil, errno
	}
	return state, nil
}

func setTerminalState(fd uintptr, state *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCSETAF),
		uintptr(unsafe.Pointer(state)),
		0,
		0,
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
