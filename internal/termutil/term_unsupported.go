//go:build !darwin && !linux && !windows

package termutil

import "errors"

func IsTerminal(fd int) bool {
	return false
}

func ReadPassword(fd int) ([]byte, error) {
	return nil, errors.New("hidden terminal input is not supported on this platform")
}
