package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Unmounter func() error

type MountOption struct {
	Source string
	Target string
	Type   string
	Flag   uintptr
	Option string
}

// Mount mounts list of mountOptions and returns a function to unmount them.
func Mount(mountOpts ...MountOption) (Unmounter, error) {
	unmounter := func() error {
		for _, p := range mountOpts {
			log.Debug().Str("source", p.Source).Str("target", p.Target).
				Msg("Unmount target")
			if err := syscall.Unmount(p.Target, 0); err != nil {
				return errors.Wrapf(err, "unable to umount %q", p.Target)
			}
		}
		return nil
	}

	for _, p := range mountOpts {
		log.Debug().Str("source", p.Source).Str("target", p.Target).
			Msg("Mount target")
		if err := syscall.Mount(p.Source, p.Target, p.Type, p.Flag, p.Option); err != nil {
			return unmounter, errors.Wrapf(err, "unable to mount %s to %s", p.Source, p.Target)
		}
	}

	return unmounter, nil
}

// OverlayMount mounts a list of source directories to a target
func OverlayMount(target string, src []string, ro bool) (Unmounter, error) {
	var upper, work []string

	if !ro {
		// Create upper and work directories for writable mount
		parentDir := filepath.Dir(strings.TrimRight(target, "/"))
		upperDir := filepath.Join(parentDir, "diff")
		workDir := filepath.Join(parentDir, "work")

		if err := os.MkdirAll(upperDir, 0700); err != nil {
			return nil, errors.Wrap(err, "can't create overlay upper directory")
		}
		if err := os.MkdirAll(workDir, 0700); err != nil {
			return nil, errors.Wrap(err, "can't create overlay work directory")
		}

		upper = append(upper, upperDir)
		work = append(work, workDir)
	}

	opt := formatOverlayFsMountOption(src, upper, work)
	newMountOption := MountOption{
		Source: "none",
		Target: target,
		Type:   "overlay",
		Flag:   0,
		Option: opt,
	}

	return Mount(newMountOption)
}

// formatOverlayFsMountOption returns formatted overlayFS mount option.
func formatOverlayFsMountOption(lowerDir, upperDir, workDir []string) string {
	lower := "lowerdir="
	lower += strings.Join(lowerDir, ":")
	upper := "upperdir="
	upper += strings.Join(upperDir, ":")
	work := "workdir="
	work += strings.Join(workDir, ":")

	opt := strings.Join([]string{lower, upper, work}, ",")
	return opt
}
