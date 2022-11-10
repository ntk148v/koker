package cgroups

import (
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/ntk148v/koker/pkg/constants"
)

type CGroups interface {
	// SetMemSwpLimit sets memory and swap limit for CGroups
	SetMemSwpLimit(memory, swap int) error
	// SetPidsLimit sets maximum processes than can be created
	SetPidsLimit(pids int) error
	// SetCPULimit  sets number of CPU for the CGroups
	SetCPULimit(cpus float64) error
	// AddProcess adds a pid into a CGroup.
	AddProcess() error
	// Remove removes CGroups
	// It will only works if there is no process running in the CGroups
	Remove()
	// GetPids returns slice of pids running on CGroups
	GetPids() ([]string, error)
}

// NewCGroups returns a new CGroups instance
func NewCGroups(path string) (CGroups, error) {
	var cg CGroups
	cgMode, err := Mode()
	if err != nil {
		return cg, err
	}

	switch cgMode {
	case Legacy, Hybrid:
		return newCGroupsv1(path)
	case Unified:
		// Handle CGroup v2
		createKokerGroup()
		return newCGroupsv2(path)
	default:
		return cg, nil
	}
}

// CGMode is the cgroups mode of the host system
type CGMode int

const (
	// Unavailable cgroup mountpoint
	Unavailable CGMode = iota
	// Legacy cgroups v1
	Legacy
	// Hybrid with cgroups v1 and v2 controllers mounted
	Hybrid
	// Unified with only cgroups v2 mounted
	Unified
)

func (c CGMode) String() string {
	switch c {
	case Legacy:
		return "Legacy"
	case Hybrid:
		return "Hybrid"
	case Unified:
		return "Unified"
	default:
		return "Unavailable"
	}
}

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
