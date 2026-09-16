//go:build linux

package utils

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// SetNamespace calls setns syscall for set of flags. It changes
// current process namespace to namespace of another process which
// can be specified by pid.
//
// NOTE: A process may not be reassociated with a new mount namespace
// if it is multi-threaded. Changing the mount namespace requires that
// the caller possess both CAP_SYS_CHROOT and CAP_SYS_ADMIN capabilities
// in its own user namespace and CAP_SYS_ADMIN in the target mount namespace.
func SetNamespace(pid string, flag int) error {
	nsBase := filepath.Join("/proc", pid, "ns")
	ns := map[int]string{
		syscall.CLONE_NEWIPC: "ipc",
		syscall.CLONE_NEWNS:  "mnt",
		syscall.CLONE_NEWNET: "net",
		syscall.CLONE_NEWPID: "pid",
		syscall.CLONE_NEWUTS: "uts",
	}

	for k, v := range ns {
		if flag&k == 0 {
			continue
		}
		nsPath := filepath.Join(nsBase, v)
		nsFile, err := os.Open(nsPath)
		if err != nil {
			return errors.Wrapf(err, "can't open %s", nsPath)
		}

		setnsErr := unix.Setns(int(nsFile.Fd()), k)
		nsFile.Close()
		if setnsErr != nil {
			return errors.Wrapf(setnsErr, "can't setns to %s", v)
		}
	}

	return nil
}
