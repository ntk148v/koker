package containers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/cgroups"
	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/filesystem"
	"github.com/ntk148v/koker/pkg/images"
	"github.com/ntk148v/koker/pkg/network"
	"github.com/ntk148v/koker/pkg/reexec"
	"github.com/ntk148v/koker/pkg/utils"
)

func ListAllContainers() ([]map[string]string, error) {
	all := make([]map[string]string, 0)
	files, err := os.ReadDir(constants.KokerContainersPath)
	if err != nil {
		return all, err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		// Load container config to retrieve config
		c, _ := NewContainer(file.Name())
		if err := c.LoadConfig(); err != nil {
			return all, err
		}

		// Get cmd
		cmd, err := c.getCmd()
		if err != nil {
			return all, err
		}

		all = append(all, map[string]string{
			"id":    c.ID,
			"image": c.Config.Image[8:],
			"cmd":   cmd,
		})
	}

	return all, nil
}

type Container struct {
	Config *v1.Config
	ID     string
	RootFS string
	log    zerolog.Logger
	cg     cgroups.CGroups
}

// NewContainer returns a new Container instance with random digest
func NewContainer(id string) (*Container, error) {
	c := &Container{
		Config: new(v1.Config),
		RootFS: filepath.Join(constants.KokerContainersPath, id, "mnt"),
		ID:     id,
		log:    log.With().Str("container", id).Logger(),
	}
	cg, err := cgroups.NewCGroups(constants.KokerApp + "/" + id)
	if err == nil {
		c.cg = cg
	}
	return c, err
}

func (c *Container) Run(src string, cmds []string, hostname string, mem, swap, pids int, cpus float64, quiet, debug bool) error {
	defer func() {
		if err := c.delete(); err != nil {
			c.log.Error().Err(err).Msg("Clean up container failed")
		}
	}()
	// Setup network
	delNet, err := c.setupNetwork(constants.KokerBridgeName)
	if err != nil {
		return errors.Wrap(err, "unable to setup network")
	}
	defer func() {
		if err := delNet(); err != nil {
			c.log.Error().Err(err).Msg("Unmount network namespace failed")
		}
	}()

	// Get image
	img, err := images.NewImage(src)
	if err != nil {
		return errors.Wrap(err, "unable to get image")
	}

	// Mount overlayfs
	unmount, err := c.mountOverlayFS(img)
	if err != nil {
		return errors.Wrap(err, "unable to mount overlayfs")
	}
	defer func() {
		if err := unmount(); err != nil {
			c.log.Error().Err(err).Msg("Unmount overlayfs failed")
		}
	}()

	// Format child options
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
	opts = append(opts, "--hostname="+hostname)
	args := append([]string{c.ID}, cmds...)
	args = append(opts, args...)
	args = append([]string{"container", "child"}, args...)
	// NOTE(kiennt26): Have to pass quiet and debug again as we re-run ourselves
	// If not set, quiet and debug mode won't work properly.
	if quiet {
		args = append([]string{"-q"}, args...)
	} else {
		if debug {
			args = append([]string{"-D"}, args...)
		}
	}

	// /proc/self/exe - a special file containing an in-memory image of the current executable.
	// In other words, we re-run ourselves, but passing childs as the first agrument.
	cmd := reexec.Command(args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID,
	}
	return cmd.Run()
}

// RunChild runs child command which is called from Run()
func (c *Container) RunChild(cmdArgs []string, hostname string, mem, swap, pids int, cpus float64) error {
	// Set hostname
	c.setHostname(hostname)

	// Setup cgroups
	if err := c.cg.AddProcess(); err != nil {
		return err
	}

	if err := c.setLimit(mem, swap, pids, cpus); err != nil {
		return errors.Wrap(err, "unable to set container's limit")
	}

	// Copy nameserver
	if err := c.copyNameServerConfig(); err != nil {
		return errors.Wrap(err, "unable to copy name server config")
	}

	// Execute command
	return c.ExecuteCommand(cmdArgs, true)
}

func (c *Container) ExecuteCommand(cmdArgs []string, child bool) error {
	c.log.Info().Msg("Execute command")
	if child {
		// Set network namespace
		unset, err := c.setNetworkNamespace()
		if err != nil {
			return errors.Wrap(err, "unable to set network namespace")
		}
		defer func() {
			if err := unset(); err != nil {
				c.log.Error().Err(err).Msg("Unset network namespace failed")
			}
		}()
	} else {
		pid, err := c.getMainPid()
		if err != nil {
			return err
		}

		// Set namespace
		if err := utils.SetNamespace(pid, syscall.CLONE_NEWUTS|syscall.CLONE_NEWIPC|syscall.CLONE_NEWPID|syscall.CLONE_NEWNET); err != nil {
			return err
		}
	}

	// Change root
	// calls chroot syscall for the given root filesystem
	if err := syscall.Chroot(c.RootFS); err != nil {
		return errors.Wrapf(err, "unable to change root to %s", c.RootFS)
	}
	// change working directory into workdir
	if c.Config.WorkingDir == "" {
		c.Config.WorkingDir = "/"
	}
	if err := os.Chdir(c.Config.WorkingDir); err != nil {
		return errors.Wrapf(err, "unable to change working directory to %s",
			c.Config.WorkingDir)
	}

	if child {
		// Mount necessaries
		mountPoints := []filesystem.MountOption{
			{Source: "tmpfs", Target: "dev", Type: "tmpfs"},
			{Source: "proc", Target: "proc", Type: "proc"},
			{Source: "sysfs", Target: "sys", Type: "sysfs"},
			{Source: "tmpfs", Target: "tmp", Type: "tmpfs"},
		}
		unmount, err := filesystem.Mount(mountPoints...)
		if err != nil {
			return err
		}
		defer func() {
			if err := unmount(); err != nil {
				c.log.Error().Err(err).Msg("Unmount mountpoints (proc, sys,tmp, dev) failed")
			}
		}()
	}

	var cmd *exec.Cmd

	if len(cmdArgs) < 1 {
		cmdArgs = append(cmdArgs, c.Config.Cmd...)
	}

	if len(c.Config.Entrypoint) > 0 {
		cmdArgs = append(c.Config.Entrypoint, cmdArgs...)
	}

	command, argv := utils.CmdAndArgs(cmdArgs)

	c.log.Debug().Str("command", command).Msg("Execute command")
	cmd = exec.Command(command, argv...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Env = c.Config.Env
	return cmd.Run()
}

// LoadConfig reads container config file
func (c *Container) LoadConfig() error {
	c.log.Debug().Msg("Load container config from file")
	conCfg := filepath.Join(constants.KokerContainersPath, c.ID, "config.json")

	file, err := os.Open(conCfg)
	if err != nil {
		return err
	}
	defer file.Close()
	configFile, err := v1.ParseConfigFile(file)
	if err != nil {
		return err
	}
	c.Config = configFile.Config.DeepCopy()
	return nil
}

// getMainPid returns the main process id
func (c *Container) getMainPid() (string, error) {
	var (
		pid string
		err error
	)
	pids, err := c.cg.GetPids()
	if err != nil {
		return pid, err
	}

	if len(pids) > 0 {
		pid = pids[len(pids)-1]
	} else {
		err = errors.New("no process id found")
	}

	return pid, err
}

// getCmd returns the running command inside container
func (c *Container) getCmd() (string, error) {
	var cmd string
	pid, err := c.getMainPid()
	if err != nil {
		return cmd, err
	}
	cmdline, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
	if err != nil {
		return cmd, err
	}
	cmd = string(bytes.TrimSpace(bytes.ReplaceAll(cmdline, []byte{0}, []byte{' '})))
	return cmd, nil
}

// copyNameServerConfig copies name resolver configurations
func (c *Container) copyNameServerConfig() error {
	c.log.Info().Msg("Copy nameserver config")
	resolvFilePaths := []string{
		fmt.Sprintf("/etc/%sresolv.conf", constants.KokerApp),
		"/etc/resolv.conf",
		"/var/run/systemd/resolve/resolv.conf",
	}
	for _, resolvFilePath := range resolvFilePaths {
		if _, err := os.Stat(resolvFilePath); os.IsNotExist(err) {
			continue
		}
		if err := utils.CopyFile(resolvFilePath,
			filepath.Join(c.RootFS, "etc/resolv.conf")); err != nil {
			return err
		}
	}
	return nil
}

// setLimit configures resource limit using cgroup
func (c *Container) setLimit(mem, swap, pids int, cpus float64) error {
	c.log.Info().Msg("Set container's limit using cgroup")
	c.log.Debug().Msg("Set container's memory limit")
	if err := c.cg.SetMemSwpLimit(mem, swap); err != nil {
		return err
	}
	c.log.Debug().Msg("Set container's pids limit")
	if err := c.cg.SetPidsLimit(pids); err != nil {
		return err
	}
	c.log.Debug().Msg("Set container's cpus limit")
	if err := c.cg.SetCPULimit(cpus); err != nil {
		return err
	}
	return nil
}

// setHostname sets container's hostname
// Default: ID[:12]
func (c *Container) setHostname(hostname string) {
	c.log.Info().Msg("Set hostname")
	c.Config.Hostname = hostname
	if c.Config.Hostname == "" {
		c.Config.Hostname = c.ID[:12]
	}
	syscall.Sethostname([]byte(c.Config.Hostname))
}

func (c *Container) delete() error {
	c.log.Info().Msg("Delete container")
	c.log.Debug().Msg("Remove container's directory")
	if err := os.RemoveAll(filepath.Join(constants.KokerContainersPath, c.ID)); err != nil {
		return errors.Wrap(err, "unable to remove container's directory")
	}
	c.log.Debug().Msg("Remove container's network namespace")
	if err := os.RemoveAll(filepath.Join(constants.KokerNetNsPath, c.ID)); err != nil {
		return errors.Wrap(err, "unable to remove network namespace")
	}
	c.log.Debug().Msg("Remove container cgroups")
	c.cg.Remove()
	return nil
}

// mountOverlayFS mounts filesystem for Container from an Image.
// It uses overlayFS for union mount of multiple layers.
func (c *Container) mountOverlayFS(img *images.Image) (filesystem.Unmounter, error) {
	c.log.Info().Str("image", img.Metadata.Name).
		Msg("Mount filesystem for container from an image")
	if err := os.MkdirAll(c.RootFS, 0700); err != nil {
		return nil, errors.Wrapf(err, "can't create %s directory", c.RootFS)
	}

	imgLayers := img.Metadata.Manifest.Layers
	layers := make([]string, 0)
	for _, i := range imgLayers {
		layers = append(layers, filepath.Join(constants.KokerImagesPath, img.Metadata.Digest, i.Digest.Hex))
	}
	unmounter, err := filesystem.OverlayMount(c.RootFS, layers, false)
	if err != nil {
		return unmounter, err
	}

	return unmounter, c.copyImageConfig(img)
}

func (c *Container) copyImageConfig(img *images.Image) error {
	c.log.Debug().Str("image", img.Metadata.Repository).Msg("Copy container config from image config")
	imgCfg := filepath.Join(constants.KokerImagesPath, img.Metadata.Digest, "config.json")
	conCfg := filepath.Join(constants.KokerContainersPath, c.ID, "config.json")
	return utils.CopyFile(imgCfg, conCfg)
}

// setupNetwork configures network for the container
func (c *Container) setupNetwork(bridge string) (filesystem.Unmounter, error) {
	c.log.Info().Msg("Setup network for container")
	nsMountTarget := filepath.Join(constants.KokerNetNsPath, c.ID)
	vethName := fmt.Sprintf("%s%.7s", constants.KokerVirtual0Pfx, c.ID)
	peerName := fmt.Sprintf("%s%.7s", constants.KokerVirtual1Pfx, c.ID)

	if err := network.SetupVirtualEthernet(vethName, peerName); err != nil {
		return nil, err
	}

	if err := network.LinkSetMaster(vethName, constants.KokerBridgeName); err != nil {
		return nil, err
	}

	unmount, err := network.MountNetNS(nsMountTarget)
	if err != nil {
		return unmount, err
	}

	if err := network.LinkSetNSByFile(nsMountTarget, peerName); err != nil {
		return unmount, err
	}

	// Change current network namespace to setup the veth
	unset, err := network.SetNetNSByFile(nsMountTarget)
	if err != nil {
		return unmount, err
	}
	defer func() {
		if err := unset(); err != nil {
			c.log.Error().Err(err).Msg("unable to unset network namespace")
		}
	}()

	ctrEthIPAddr := utils.GenIPAddress()
	if err := network.LinkRename(peerName, constants.KokerCtrEthName); err != nil {
		return unmount, err
	}
	if err := network.LinkAddAddr(constants.KokerCtrEthName, ctrEthIPAddr); err != nil {
		return unmount, err
	}
	if err := network.LinkSetup(constants.KokerCtrEthName); err != nil {
		return unmount, err
	}
	if err := network.LinkAddGateway(constants.KokerCtrEthName, constants.KokerBridgeDefaultIP); err != nil {
		return unmount, err
	}
	if err := network.LinkSetup("lo"); err != nil {
		return unmount, err
	}

	return unmount, nil
}

// setNetworkNamespace
func (c *Container) setNetworkNamespace() (network.Unsetter, error) {
	c.log.Debug().Msg("Set network namespace")
	netns := filepath.Join(constants.KokerNetNsPath, c.ID)
	return network.SetNetNSByFile(netns)
}
