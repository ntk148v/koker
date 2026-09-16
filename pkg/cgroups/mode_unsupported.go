//go:build !linux

package cgroups

import "errors"

// Mode returns the cgroups mode running on the host
func Mode() (CGMode, error) {
	return Unavailable, errors.New("cgroups are only supported on Linux")
}
