package network

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/ntk148v/koker/pkg/filesystem"
)

type Unsetter func() error

// MountNetNS creates and mounts a new network namespace, then return a
// function to unmount
func MountNetNS(nsTarget string) (filesystem.Unmounter, error) {
	log.Info().Str("netns", nsTarget).Msg("Mount new network namespace")
	_, err := os.OpenFile(nsTarget, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_EXCL, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create target file")
	}

	// store current network namespace
	file, err := os.OpenFile("/proc/self/ns/net", os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	log.Debug().Str("netns", nsTarget).Msg("Call syscall unshare CLONE_NEWNET")
	if err := syscall.Unshare(syscall.CLONE_NEWNET); err != nil {
		return nil, errors.Wrap(err, "unshare syscall failed")
	}
	mountPoint := filesystem.MountOption{
		Source: "/proc/self/ns/net",
		Target: nsTarget,
		Type:   "bind",
		Flag:   syscall.MS_BIND,
	}
	log.Debug().Str("netns", nsTarget).Msg("Mount new network namespace")
	unmount, err := filesystem.Mount(mountPoint)
	if err != nil {
		return unmount, err
	}

	// reset previous network namespace
	log.Debug().Str("netns", nsTarget).Msg("Set network namespace")
	if err := unix.Setns(int(file.Fd()), syscall.CLONE_NEWNET); err != nil {
		return unmount, errors.Wrap(err, "setns syscall failed: ")
	}

	return unmount, nil
}

func SetNetNSByFile(filename string) (Unsetter, error) {
	log.Info().Str("netns", filename).Msg("Set network namespace by file")
	currentNS, err := os.OpenFile("/proc/self/ns/net", os.O_RDONLY, 0)
	unsetFunc := func() error {
		defer currentNS.Close()
		if err != nil {
			return err
		}
		return unix.Setns(int(currentNS.Fd()), syscall.CLONE_NEWNET)
	}

	netnsFile, err := os.OpenFile(filename, syscall.O_RDONLY, 0)
	if err != nil {
		return unsetFunc, errors.Wrap(err, "unable to open network namespace file")
	}
	defer netnsFile.Close()
	if err := unix.Setns(int(netnsFile.Fd()), syscall.CLONE_NEWNET); err != nil {
		return unsetFunc, errors.Wrap(err, "unset syscall failed")
	}
	return unsetFunc, err
}

// LinkSetNSByFile puts link device into a new network namespace
func LinkSetNSByFile(filename, linkName string) error {
	log.Info().Str("netns", filename).Str("link", linkName).
		Msg("Put link device into a new network namespace")
	netnsFile, err := os.OpenFile(filename, syscall.O_RDONLY, 0)
	if err != nil {
		return errors.Wrap(err, "unable to open netns file")
	}
	defer netnsFile.Close()

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}
	return netlink.LinkSetNsFd(link, int(netnsFile.Fd()))
}
