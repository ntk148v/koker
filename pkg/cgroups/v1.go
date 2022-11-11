package cgroups

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

type cgroupsv1 struct {
	dirs map[string]string
}

func newCGroupsv1(path string) (cgroupsv1, error) {
	cg := cgroupsv1{}
	// create dirs
	cg.dirs = map[string]string{
		"memory": filepath.Join(constants.CGroupMountpoint, "memory", path),
		"cpu":    filepath.Join(constants.CGroupMountpoint, "cpu", path),
		"pids":   filepath.Join(constants.CGroupMountpoint, "pids", path),
	}

	for _, dir := range cg.dirs {
		if err := utils.CreateDir(dir); err != nil {
			return cg, err
		}
	}
	return cg, nil
}

// SetMemSwpLimit sets memory and swap limit for CGroups
func (cg cgroupsv1) SetMemSwpLimit(memory, swap int) error {
	if memory > 0 {
		memFile := filepath.Join(cg.dirs["memory"], "memory.limit_in_bytes")
		if err := ioutil.WriteFile(memFile, []byte(strconv.Itoa(memory*1024*1024)), 0644); err != nil {
			return err
		}
		if swap > 0 {
			memswFile := filepath.Join(cg.dirs["memory"], "memory.memsw.limit_in_bytes")
			if err := ioutil.WriteFile(memswFile, []byte(strconv.Itoa((memory+swap)*1024*1024)), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetPidsLimit sets maximum processes than can be created
// simultaneously in CGroups
func (cg cgroupsv1) SetPidsLimit(pids int) error {
	if pids > 0 {
		pidsFile := filepath.Join(cg.dirs["pids"], "pids.max")
		if err := ioutil.WriteFile(pidsFile, []byte(strconv.Itoa(pids)), 0644); err != nil {
			return err
		}
	}
	return nil
}

// SetCPULimit sets number of CPU for the CGroups
func (cg cgroupsv1) SetCPULimit(cpus float64) error {
	if int(cpus) > 0 && int(cpus) < runtime.NumCPU() {
		cpuPeriodFile := filepath.Join(cg.dirs["cpu"], "cpu.cfs_period_us")
		if err := ioutil.WriteFile(cpuPeriodFile, []byte(strconv.Itoa(constants.DefaultCfsPeriod)),
			0644); err != nil {
			return err
		}

		cpuQuotaFile := filepath.Join(cg.dirs["cpu"], "cpu.cfs_quota_us")
		if err := ioutil.WriteFile(cpuQuotaFile, []byte(strconv.Itoa(int(cpus*constants.DefaultCfsPeriod))),
			0644); err != nil {
			return err
		}
	}
	return nil
}

// AddProcess adds pids into a CGroup
func (cg cgroupsv1) AddProcess() error {
	// Get pid
	pid := os.Getpid()
	for _, dir := range cg.dirs {
		procsFile := filepath.Join(dir, "cgroup.procs")
		if err := ioutil.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0700); err != nil {
			return err
		}
	}
	return nil
}

// Remove removes CGroups
// It will only works if there is no process running in the CGroups
func (cg cgroupsv1) Remove() {
	for _, dir := range cg.dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		os.Remove(dir)
	}
}

// GetPids returns slice of pids running on CGroups
func (cg cgroupsv1) GetPids() ([]string, error) {
	var pids []string
	procsFile, err := os.Open(filepath.Join(cg.dirs["pids"],
		"cgroup.procs"))
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
