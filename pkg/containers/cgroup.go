package containers

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

type cgroups struct {
	// CGroups absolute file path
	memPath       string
	memswPath     string
	cpuPeriodPath string
	cpuQuotaPath  string
	pidsPath      string
	mem           int
	memsw         int
	cfsPeriod     int
	cfsQuota      float64
	pids          int
}

// newCGroup cretates an empty cgroups
func newCGroup(path string) *cgroups {
	// create dirs
	dirs := map[string]string{
		"memory": filepath.Join(constants.CGroupPath, "memory", path),
		"cpu":    filepath.Join(constants.CGroupPath, "cpu", path),
		"pids":   filepath.Join(constants.CGroupPath, "pids", path),
	}

	for _, dir := range dirs {
		utils.CreateDir(dir)
	}

	return &cgroups{
		memPath:       filepath.Join(dirs["memory"], constants.MemLimitFilename),
		memswPath:     filepath.Join(dirs["memory"], constants.MemswLimitFilename),
		cpuPeriodPath: filepath.Join(dirs["cpu"], constants.CpuPeriodFilename),
		cpuQuotaPath:  filepath.Join(dirs["cpu"], constants.CpuQuotaFilename),
		pidsPath:      filepath.Join(dirs["pids"], constants.MaxProcessFilename),
	}
}

// setMemSwpLimit sets memory and swap limit for CGroups
func (cg *cgroups) setMemSwpLimit(memory, swap int) error {
	if memory > 0 {
		cg.mem = memory
		if err := ioutil.WriteFile(cg.memPath, []byte(strconv.Itoa(cg.mem*1024*1024)),
			0644); err != nil {
			return err
		}
		if swap > 0 {
			cg.memsw = swap
			if err := ioutil.WriteFile(cg.memswPath,
				[]byte(strconv.Itoa((cg.mem+cg.memsw)*1024*1024)), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// setPidsLimit sets maximum processes than can be created
// simultaneously in CGroups
func (cg *cgroups) setPidsLimit(pids int) error {
	if pids > 0 {
		cg.pids = pids
		if err := ioutil.WriteFile(cg.pidsPath, []byte(strconv.Itoa(cg.pids)), 0644); err != nil {
			return err
		}
	}
	return nil
}

// setCPULimit sets number of CPU for the CGroups
func (cg *cgroups) setCPULimit(cpus float64) error {
	if int(cpus) > 0 && int(cpus) < runtime.NumCPU() {
		cg.cfsPeriod = constants.DefaultCfsPeriod
		if err := ioutil.WriteFile(cg.cpuPeriodPath, []byte(strconv.Itoa(cg.cfsPeriod)),
			0644); err != nil {
			return err
		}
		cg.cfsQuota = constants.DefaultCfsPeriod * cpus
		if err := ioutil.WriteFile(cg.cpuQuotaPath, []byte(strconv.Itoa(int(cg.cfsQuota))),
			0644); err != nil {
			return err
		}
	}
	return nil
}

// addProcess adds a pid into a CGroup.
func (cg *cgroups) addProcess() error {
	// Get pid
	pid := os.Getpid()
	dirs := []string{
		filepath.Dir(cg.memPath), filepath.Dir(cg.memswPath),
		filepath.Dir(cg.cpuPeriodPath), filepath.Dir(cg.cpuQuotaPath),
		filepath.Dir(cg.pidsPath),
	}

	for _, dir := range dirs {
		if err := ioutil.WriteFile(filepath.Join(dir, constants.ProcsFilename),
			[]byte(strconv.Itoa(pid)), 0700); err != nil {
			return err
		}
	}
	return nil
}

// Remove removes CGroups
// It will only works if there is no process running in the CGroups
func (cg *cgroups) remove() {
	dirs := []string{
		filepath.Dir(cg.memPath), filepath.Dir(cg.memswPath),
		filepath.Dir(cg.cpuPeriodPath), filepath.Dir(cg.cpuQuotaPath),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		os.Remove(dir)
	}
}

// getPids returns slice of pids running on CGroups
func (cg *cgroups) getPids() ([]string, error) {
	var pids []string
	procFile, err := os.Open(filepath.Join(filepath.Dir(cg.pidsPath),
		constants.ProcsFilename))
	if err != nil {
		return pids, err
	}
	defer procFile.Close()

	scanner := bufio.NewScanner(procFile)
	for scanner.Scan() {
		pid := scanner.Text()
		pids = append(pids, pid)
	}

	return pids, nil
}
