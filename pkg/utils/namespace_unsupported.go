//go:build !linux

package utils

import "errors"

// SetNamespace is only supported on Linux.
func SetNamespace(pid string, flag int) error {
	return errors.New("SetNamespace is only supported on Linux")
}
