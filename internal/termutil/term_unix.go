//go:build darwin || linux

package termutil

import (
	"syscall"
	"unsafe"
)

func IsTerminal(fd int) bool {
	_, err := getTermios(fd)
	return err == nil
}

func ReadPassword(fd int) ([]byte, error) {
	termios, err := getTermios(fd)
	if err != nil {
		return nil, err
	}

	newState := *termios
	newState.Lflag &^= syscall.ECHO
	newState.Lflag |= syscall.ICANON | syscall.ISIG
	newState.Iflag |= syscall.ICRNL
	if err := setTermios(fd, &newState); err != nil {
		return nil, err
	}
	defer setTermios(fd, termios)

	return readPasswordLine(passwordReader(fd))
}

func getTermios(fd int) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctlReadTermios),
		uintptr(unsafe.Pointer(termios)),
		0,
		0,
		0,
	)
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd int, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctlWriteTermios),
		uintptr(unsafe.Pointer(termios)),
		0,
		0,
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

type passwordReader int

func (r passwordReader) Read(buf []byte) (int, error) {
	return syscall.Read(int(r), buf)
}
