package filesystem

import (
	"strings"
)

type Unmounter func() error

type MountOption struct {
	Source string
	Target string
	Type   string
	Flag   uintptr
	Option string
}

// formatOverlayFsMountOption returns formatted overlayFS mount option.
func formatOverlayFsMountOption(lowerDir, upperDir, workDir []string) string {
	opts := []string{"lowerdir=" + strings.Join(lowerDir, ":")}
	if len(upperDir) > 0 {
		opts = append(opts, "upperdir="+strings.Join(upperDir, ":"))
	}
	if len(workDir) > 0 {
		opts = append(opts, "workdir="+strings.Join(workDir, ":"))
	}
	return strings.Join(opts, ",")
}
