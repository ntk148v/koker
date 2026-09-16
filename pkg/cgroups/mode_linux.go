//go:build linux

package cgroups

import (
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/ntk148v/koker/pkg/constants"
)

// Mode returns the cgroups mode running on the host
func Mode() (CGMode, error) {
	var (
		st     unix.Statfs_t
		cgMode CGMode
	)
	if err := unix.Statfs(constants.CGroupMountpoint, &st); err != nil {
		return Unavailable, err
	}
	switch st.Type {
	case unix.CGROUP2_SUPER_MAGIC:
		cgMode = Unified
	default:
		cgMode = Legacy
		if err := unix.Statfs(filepath.Join(constants.CGroupMountpoint, "unified"), &st); err != nil {
			return Unavailable, err
		}
		if st.Type == unix.CGROUP2_SUPER_MAGIC {
			cgMode = Hybrid
		}
	}
	return cgMode, nil
}
