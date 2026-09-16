//go:build !linux

package filesystem

import "errors"

// Mount is only supported on Linux.
func Mount(mountOpts ...MountOption) (Unmounter, error) {
	return nil, errors.New("mount is only supported on Linux")
}

// OverlayMount is only supported on Linux.
func OverlayMount(target string, src []string, ro bool) (Unmounter, error) {
	return nil, errors.New("overlay mount is only supported on Linux")
}
