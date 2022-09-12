package containers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/images"
	"github.com/ntk148v/koker/pkg/network"
	"github.com/ntk148v/koker/pkg/utils"
)

// createConDirs creates containers' directories
func createConDirs(conID string) error {
	conHome := filepath.Join(constants.KokerContainersPath, conID)
	conDirs := []string{
		filepath.Join(conHome, "fs"),
		filepath.Join(conHome, "fs", "mnt"),
		filepath.Join(conHome, "fs", "upperdir"),
		filepath.Join(conHome, "fs", "workdir"),
	}
	for _, dir := range conDirs {
		if err := utils.CreateDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func mountOverlayFS(conID, imgSHA string) error {
	var srcLayers []string
	imageBasePath := filepath.Join(constants.KokerImagesPath,
		imgSHA)
	manifestJSON := filepath.Join(imageBasePath, "manifest.json")
	// Get manifest
	m := images.Manifest{}
	if err := images.ParseManifest(manifestJSON, &m); err != nil {
		return err
	}
	if len(m) == 0 || len(m[0].Layers) == 0 {
		return errors.New("could not find any layers")
	} else if len(m) > 1 {
		return errors.New("unexpected mutiple manifestes")
	}

	for _, layer := range m[0].Layers {
		srcLayers = append([]string{filepath.Join(imageBasePath, layer[:12], "fs")}, srcLayers...)
	}

	conHome := filepath.Join(constants.KokerContainersPath, conID, "fs")
	mntOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(srcLayers, ":"),
		filepath.Join(conHome, "upperdir"),
		filepath.Join(conHome, "workdir"))
	return unix.Mount("none", filepath.Join(conHome, "mnt"), "overlay", 0, mntOpts)
}

func initContainer(conID, imgSHA string, cmdArgs []string, mem, pids int, cpus float64) error {
	log.Info().Str("containerid", conID).
		Msg("Setup network namespace")

	if err := network.SetupNetNS(conID); err != nil {
		log.Error().Err(err).Str("containerid", conID).
			Msg("Unable to setup network namespace")
		return err
	}

	// Setup virtual interfaces
	log.Info().Str("containerid", conID).
		Msg("Setup virtual interfaces")
	if err := network.SetupConNetInf(conID); err != nil {
		log.Error().Err(err).Str("containerid", conID).
			Msg("Unable to setup virtual interfaces")
		return err
	}

	// Namespace
	log.Info().Str("containerid", conID).
		Msg("Init container and execute command")

	var opts []string
	if mem > 0 {
		opts = append(opts, "--mem="+strconv.Itoa(mem))
	}
	if pids > 0 {
		opts = append(opts, "--pids="+strconv.Itoa(pids))
	}
	if cpus > 0 {
		opts = append(opts, "--cpus="+strconv.FormatFloat(cpus, 'f', 1, 64))
	}
	args := append([]string{conID, imgSHA}, cmdArgs...)
	args = append(opts, args...)
	args = append([]string{"container", "child"}, args...)
	// /proc/self/exe - a special file containing an in-memory image of the current executable.
	// In other words, we re-run ourselves, but passing childs as the first agrument.
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Cloneflags is only available in Linux
	// Check here: https://en.wikipedia.org/wiki/Linux_namespaces#Namespace_kinds
	// CLONE_NEWUTS namespace isolates hostname
	// CLONE_NEWPID namespace isolates processes
	// CLONE_NEWNS namespace isolates mounts
	// CLONE_NEWIPC namespace isolates interprocess communication (IPC)
	// CLONE_NEWNET namespace isolates network
	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: unix.CLONE_NEWPID |
			unix.CLONE_NEWNS |
			unix.CLONE_NEWUTS |
			unix.CLONE_NEWIPC,
	}

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Str("containerid", conID).
			Msg("Unable to create container")
		return err
	}

	return nil
}

func parseContainerCfg(conID string) (v1.Config, error) {
	var conCfg v1.Config
	conCfgPath := filepath.Join(constants.KokerContainersPath, conID, "config.json")
	f, err := os.Open(conCfgPath)
	defer f.Close()
	if err != nil {
		return conCfg, err
	}
	cfg, err := v1.ParseConfigFile(f)
	conCfg = *cfg.Config.DeepCopy()
	return conCfg, nil
}

// joinConNetNs
func joinConNetNs(conID string) error {
	nsMount := filepath.Join(constants.KokerNetNsPath, conID)
	fd, err := unix.Open(nsMount, unix.O_RDONLY, 0)
	if err != nil {
		log.Error().Err(err).Msg("Unable to open file")
		return err
	}

	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Error().Err(err).Msg("Setns system call failed")
		return err
	}
	return nil
}

func copyNameserverCfg(conID string) error {
	resolvFilePaths := []string{
		"/var/run/systemd/resolve/resolv.conf",
		"/etc/gockerresolv.conf",
		"/etc/resolv.conf",
	}
	for _, resolvFilePath := range resolvFilePaths {
		if _, err := os.Stat(resolvFilePath); os.IsNotExist(err) {
			continue
		} else {
			return utils.CopyFile(resolvFilePath,
				filepath.Join(constants.KokerContainersPath, conID,
					"/fs/mnt/etc/resolv.conf"))
		}
	}
	return nil
}

// ExecuteContainerCommand
// TODO(kiennt26): Add error logging later
func ExecuteContainerCommand(conID, imgSHA string, cmdArgs []string, mem, pids int, cpus float64) error {
	mntPath := filepath.Join(constants.KokerContainersPath, conID, "fs/mnt")

	// Parse container config
	log.Debug().Str("containerid", conID).
		Msg("Parse container config")
	conCfg, err := parseContainerCfg(conID)
	if err != nil {
		return err
	}

	// Construct commands
	var cmd *exec.Cmd

	if len(cmdArgs) < 1 {
		if len(conCfg.Entrypoint) > 0 {
			cmdArgs = append(cmdArgs, conCfg.Entrypoint...)
		}
		cmdArgs = append(cmdArgs, conCfg.Cmd...)
	}

	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Set hostname
	log.Debug().Str("containerid", conID).
		Msg("Set hostname")
	if err := unix.Sethostname([]byte(conID)); err != nil {
		return err
	}

	// Join container network namespace
	log.Debug().Str("containerid", conID).
		Msg("Join container network namespace")
	if err := joinConNetNs(conID); err != nil {
		return err
	}

	// Create CGroup directories
	log.Debug().Str("containerid", conID).
		Msg("Create CGroup directories")
	if err := createCGroups(conID); err != nil {
		return err
	}

	// Configure CGroup
	log.Debug().Str("containerid", conID).
		Msg("Configure CGroup (memory, cpu, pids)")
	if err := configCGroup(conID, mem, pids, cpus); err != nil {
		return err
	}

	// Copy nameserver
	log.Debug().Str("containerid", conID).
		Msg("Copy nameserver")
	if err := copyNameserverCfg(conID); err != nil {
		return err
	}

	// Chroot
	log.Debug().Str("containerid", conID).
		Msg("Do chroot")
	if err := unix.Chroot(mntPath); err != nil {
		return err
	}
	// Change directory
	if err := os.Chdir("/"); err != nil {
		return err
	}

	// Mount filesystem
	log.Debug().Str("containerid", conID).
		Msg("Mount filesystem")
	for _, dir := range []string{"/proc", "/sys", "/dev/pts"} {
		// handle error later
		utils.CreateDir(dir)
	}
	unix.Mount("proc", "/proc", "proc", 0, "")
	unix.Mount("tmpfs", "/tmp", "tmpfs", 0, "")
	unix.Mount("tmpfs", "/dev", "tmpfs", 0, "")
	unix.Mount("devpts", "/dev/pts", "devpts", 0, "")
	unix.Mount("sysfs", "/sys", "sysfs", 0, "")

	// Setup local interface
	log.Debug().Str("containerid", conID).
		Msg("Setup local interface")
	network.SetupLocalInterface()

	defer func() {
		unix.Unmount("/dev/pts", 0)
		unix.Unmount("/dev", 0)
		unix.Unmount("/sys", 0)
		unix.Unmount("/proc", 0)
		unix.Unmount("/tmp", 0)
	}()
	cmd.Env = conCfg.Env
	return cmd.Run()
}

func unmountNetNS(conID string) {
	netNSPath := filepath.Join(constants.KokerNetNsPath, conID)
	if _, err := os.Stat(netNSPath); os.IsNotExist(err) {
		return
	}
	if err := unix.Unmount(netNSPath, 0); err != nil {
		log.Warn().Err(err).Str("containerid", conID).
			Str("netns", netNSPath).Msg("Unable to unmount network namespace")
	}
}

func unmountConFS(conID string) {
	mountedPath := filepath.Join(constants.KokerContainersPath, conID, "fs", "mnt")
	if _, err := os.Stat(mountedPath); os.IsNotExist(err) {
		return
	}
	if err := unix.Unmount(mountedPath, 0); err != nil {
		log.Warn().Err(err).Str("containerid", conID).
			Str("mounted", mountedPath).Msg("Unable to unmount container filesystem")
	}
}

func InitContainer(img string, cmds []string, mem, pids int, cpus float64) error {
	containerID := utils.GenUID()
	imageSHA, err := images.DownloadImage(img)
	if err != nil {
		return err
	}

	defer func() {
		// Clean up
		log.Debug().Str("containerid", containerID).
			Msg("Cleanup, remove directories and stuffs")
		unmountNetNS(containerID)
		unmountConFS(containerID)
		removeCGroup(containerID)
		os.RemoveAll(filepath.Join(constants.KokerContainersPath, containerID))
	}()

	// Create container's directories
	log.Debug().Str("containerid", containerID).
		Msg("Create container's directories")
	if err := createConDirs(containerID); err != nil {
		log.Error().Err(err).Str("containerid", containerID).
			Msg("Unable to create container's directories")
		return err
	}

	// Mount overlay filesystem
	log.Debug().Str("containerid", containerID).
		Str("imageSHA", imageSHA).
		Msg("Mount overlay filesystem from image")
	if err := mountOverlayFS(containerID, imageSHA); err != nil {
		log.Error().Err(err).Str("containerid", containerID).
			Msg("Unable to mount overlay filesystem")
		return err
	}

	// Copy image config
	log.Debug().Str("containerid", containerID).
		Str("imageSHA", imageSHA).
		Msg("Generate container config from image config")
	imgCfgPath := filepath.Join(constants.KokerImagesPath, imageSHA, imageSHA+".json")
	conCfgPath := filepath.Join(constants.KokerContainersPath, containerID, "config.json")
	if err := utils.CopyFile(imgCfgPath, conCfgPath); err != nil {
		return err
	}

	// Setup virtual ethernet on host
	log.Debug().Str("containerid", containerID).
		Msg("Setup virtual ethernet on host")
	if err := network.SetupVirtEth(containerID); err != nil {
		log.Error().Err(err).Str("containerid", containerID).
			Msg("Unable to setup virtual ethernet on host")
		return err
	}

	// Init container and excute command
	log.Debug().Str("containerid", containerID).
		Msg("Init container and execute command")
	if err := initContainer(containerID, imageSHA, cmds, mem, pids, cpus); err != nil {
		log.Error().Err(err).Str("containerid", containerID).
			Msg("Unable to create a container")
		return err
	}

	log.Info().Str("containerid", containerID).
		Msg("Container did a good job, bye bye")

	return nil
}
