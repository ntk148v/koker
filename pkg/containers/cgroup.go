package containers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

// createCGroups creats CGroup directories
func createCGroups(conID string) error {
	cgroups := []string{"/sys/fs/cgroup/memory/" + constants.KokerApp + conID,
		"/sys/fs/cgroup/pids/" + constants.KokerApp + conID,
		"/sys/fs/cgroup/cpu/" + constants.KokerApp + conID}
	for _, dir := range cgroups {
		if err := utils.CreateDir(dir); err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.Join(dir, "notify_on_release"),
			[]byte("1"), 0700); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(dir, "cgroup.procs"),
			[]byte(strconv.Itoa(os.Getpid())), 0700); err != nil {
			return err
		}
	}

	return nil
}

func removeCGroup(conID string) {
	cgroups := []string{"/sys/fs/cgroup/memory/" + constants.KokerApp + conID,
		"/sys/fs/cgroup/pids/" + constants.KokerApp + conID,
		"/sys/fs/cgroup/cpu/" + constants.KokerApp + conID}

	for _, dir := range cgroups {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		os.Remove(dir)
	}
}

// configCGroup
func configCGroup(conID string, mem, pids int, cpus float64) error {
	if mem > 0 {
		if err := setMemLimit(conID, mem); err != nil {
			return err
		}
	}

	if pids > 0 {
		if err := setPidsLimit(conID, pids); err != nil {
			return err
		}
	}

	if cpus > 0 {
		if err := setCPUsLimit(conID, cpus); err != nil {
			return err
		}
	}

	return nil
}

func setMemLimit(conID string, mem int) error {
	memFilePath := filepath.Join("/sys/fs/cgroup/memory", constants.KokerApp,
		conID, "memory.limit_in_bytes")
	return ioutil.WriteFile(memFilePath, []byte(strconv.Itoa(mem*1024*1024)), 0644)
}

func setPidsLimit(conID string, pids int) error {
	maxProcsPath := filepath.Join("/sys/fs/cgroup/pids", constants.KokerApp,
		conID, "pids.max")
	return ioutil.WriteFile(maxProcsPath, []byte(strconv.Itoa(pids)), 0644)
}

func setCPUsLimit(conID string, cpus float64) error {
	cfsPeriodPath := filepath.Join("/sys/fs/cgroup/cpu", constants.KokerApp,
		conID, "cpu.cfs_period_us")
	cfsQuotaPath := filepath.Join("/sys/fs/cgroup/cpu", constants.KokerApp,
		conID, "cpu.cfs_quota_us")

	if err := ioutil.WriteFile(cfsPeriodPath, []byte(strconv.Itoa(1000000)), 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(cfsQuotaPath, []byte(strconv.Itoa(int(1000000*cpus))), 0644)
}
