package containers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	log.Debug().Str("containerid", conID).
		Msg("Setup network namespace")

	if err := network.SetupNetNS(conID); err != nil {
		log.Error().Err(err).Str("containerid", conID).
			Msg("Unable to setup network namespace")
		return err
	}

	// Setup virtual interface
	log.Debug().Str("containerid", conID).
		Msg("Setup virtual interface")
	if err := network.SetupConNetInf(conID); err != nil {
		log.Error().Err(err).Str("containerid", conID).
			Msg("Unable to setup virtual interfaces")
		return err
	}

	// Namespace
	log.Debug().Str("containerid", conID).
		Msg("Setup other namespaces")
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
	opts = append(opts, "--img="+imgSHA)
	args := append([]string{conID}, cmdArgs...)
	args = append(opts, args...)
	args = append([]string{"child"}, args...)
	// /proc/self/exe - a special file containing an in-memory image of the current executable.
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
