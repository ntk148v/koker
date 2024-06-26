package cgroups

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

// createKokerGroup creates a child group of Root
// then enable cpu, memory, and pids controllers
func createKokerGroup() error {
	kokerCGroup := filepath.Join(constants.CGroupMountpoint, constants.KokerApp)
	if err := utils.CreateDir(kokerCGroup); err != nil {
		return err
	}

	// Enable controllers
	return os.WriteFile(filepath.Join(kokerCGroup, "cgroup.subtree_control"),
		[]byte("+cpu +memory +pids"), 0644)
}

type cgroupsv2 struct {
	dir string
}

func newCGroupsv2(path string) (cgroupsv2, error) {
	cg := cgroupsv2{
		dir: filepath.Join(constants.CGroupMountpoint, path),
	}

	return cg, utils.CreateDir(cg.dir)
}

// SetMemSwpLimit sets memory and swap limit for CGroups
func (cg cgroupsv2) SetMemSwpLimit(memory, swap int) error {
	if memory > 0 {
		memFile := filepath.Join(cg.dir, "memory.max")
		if err := os.WriteFile(memFile, []byte(strconv.Itoa(memory*1024*1024)), 0644); err != nil {
			return err
		}
		if swap > 0 {
			memswFile := filepath.Join(cg.dir, "memory.swap.max")
			if err := os.WriteFile(memswFile, []byte(strconv.Itoa((memory+swap)*1024*1024)), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetPidsLimit sets maximum processes than can be created
// simultaneously in CGroups
func (cg cgroupsv2) SetPidsLimit(pids int) error {
	if pids > 0 {
		pidsFile := filepath.Join(cg.dir, "pids.max")
		if err := os.WriteFile(pidsFile, []byte(strconv.Itoa(pids)), 0644); err != nil {
			return err
		}
	}
	return nil
}

// SetCPULimit sets number of CPU for the CGroups
func (cg cgroupsv2) SetCPULimit(cpus float64) error {
	if int(cpus) > 0 && int(cpus) < runtime.NumCPU() {
		cpuFile := filepath.Join(cg.dir, "cpu.max")
		cpuVal := fmt.Sprintf("%d%d", int(cpus*constants.DefaultCfsPeriod),
			constants.DefaultCfsPeriod)
		if err := os.WriteFile(cpuFile, []byte(cpuVal), 0644); err != nil {
			return err
		}
	}
	return nil
}

// AddProcess adds pids into a CGroup
func (cg cgroupsv2) AddProcess() error {
	// Get pid
	pid := os.Getpid()
	procsFile := filepath.Join(cg.dir, "cgroup.procs")
	return os.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0700)
}

// Remove removes CGroups
// It will only works if there is no process running in the CGroups
func (cg cgroupsv2) Remove() {
	os.Remove(cg.dir)
}

// GetPids returns slice of pids running on CGroups
func (cg cgroupsv2) GetPids() ([]string, error) {
	var pids []string
	procsFile, err := os.Open(filepath.Join(cg.dir, "cgroup.procs"))
	if err != nil {
		return pids, err
	}
	defer procsFile.Close()

	scanner := bufio.NewScanner(procsFile)
	for scanner.Scan() {
		pid := scanner.Text()
		pids = append(pids, pid)
	}

	return pids, nil
}
