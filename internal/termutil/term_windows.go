//go:build windows

package termutil

import (
	"os"
	"syscall"
)

const (
	enableProcessedInput       = 0x0001
	enableLineInput            = 0x0002
	enableEchoInput            = 0x0004
	enableProcessedOutput      = 0x0001
	enableVirtualTerminalInput = 0x0200
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func IsTerminal(fd int) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(fd), &mode) == nil
}

func ReadPassword(fd int) ([]byte, error) {
	handle := syscall.Handle(fd)

	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return nil, err
	}

	nextMode := mode
	nextMode &^= enableEchoInput | enableLineInput
	nextMode |= enableProcessedInput | enableProcessedOutput | enableVirtualTerminalInput
	if err := setConsoleMode(handle, nextMode); err != nil {
		return nil, err
	}
	defer setConsoleMode(handle, mode)

	return readPasswordLine(os.NewFile(uintptr(fd), "stdin"))
}

func setConsoleMode(handle syscall.Handle, mode uint32) error {
	r1, _, err := procSetConsoleMode.Call(uintptr(handle), uintptr(mode))
	if r1 != 0 {
		return nil
	}
	if err != syscall.Errno(0) {
		return err
	}
	return syscall.EINVAL
}
