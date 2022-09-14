package containers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/filesystem"
	"github.com/ntk148v/koker/pkg/images"
	"github.com/ntk148v/koker/pkg/network"
	"github.com/ntk148v/koker/pkg/utils"
)

type Container struct {
	Config *v1.Config
	ID     string
	RootFS string
	Pids   []int
	log    zerolog.Logger
	cg     *cgroups
}

// NewContainer returns a new Container instance with random digest
func NewContainer(id string) *Container {
	return &Container{
		Config: new(v1.Config),
		RootFS: filepath.Join(constants.KokerContainersPath, id, "mnt"),
		ID:     id,
		log:    log.With().Str("container", id).Logger(),
		cg:     newCGroup(filepath.Join(constants.KokerApp, id)),
	}
}

func (c *Container) Run(src string, cmds []string, mem, swap, pids int, cpus float64) error {
	// Setup network
	delNet, err := c.SetupNetwork(constants.KokerBridgeName)
	if err != nil {
		return errors.Wrap(err, "unable to setup network")
	}
	defer delNet()

	// Get image
	img, err := images.NewImage(src)
	if err != nil {
		return errors.Wrap(err, "unable to get image")
	}
	// Check image exist
	if _, exist := images.GetImage(img.ID); !exist {
		if err := img.Download(); err != nil {
			return errors.Wrap(err, "unable to download image's layers")
		}
	}

	// Mount overlayfs
	unmount, err := c.MountOverlayFS(img)
	if err != nil {
		return errors.Wrap(err, "unable to mount overlayfs")
	}
	defer unmount()

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
	args := append([]string{c.ID, img.ID}, cmds...)
	args = append(opts, args...)
	args = append([]string{"container", "child"}, args...)
	// /proc/self/exe - a special file containing an in-memory image of the current executable.
	// In other words, we re-run ourselves, but passing childs as the first agrument.
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (c *Container) ExecuteCommand(imgSHA string, cmdArgs []string, mem, swap, pids int, cpus float64) error {
	defer c.delete()
	c.SetHostname()
	// Set network
	unset, err := c.SetNetworkNamespace()
	if err != nil {
		return errors.Wrap(err, "unable to set network namespace")
	}
	defer unset()

	// Setup cgroups
	if err := c.SetLimit(mem, swap, pids, cpus); err != nil {
		return errors.Wrap(err, "unable to set container's limit")
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

	// Mount necessaries
	mountPoints := []filesystem.MountOption{
		{Source: "proc", Target: "proc", Type: "proc"},
		{Source: "sysfs", Target: "sys", Type: "sysfs"},
	}
	unmount, err := filesystem.Mount(mountPoints...)
	if err != nil {
		return err
	}
	defer unmount()

	var cmd *exec.Cmd

	if len(cmdArgs) < 1 {
		if len(c.Config.Entrypoint) > 0 {
			cmdArgs = append(cmdArgs, c.Config.Entrypoint...)
		}
		cmdArgs = append(cmdArgs, c.Config.Cmd...)
	}

	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Env = c.Config.Env

	return cmd.Run()
}

func (c *Container) SetLimit(mem, swap, pids int, cpus float64) error {
	c.log.Info().Msg("Set container's limit using cgroup")
	c.log.Debug().Msg("Set container's memory limit")
	if err := c.cg.setMemSwpLimit(mem, swap); err != nil {
		return err
	}
	c.log.Debug().Msg("Set container's pids limit")
	if err := c.cg.setPidsLimit(pids); err != nil {
		return err
	}
	c.log.Debug().Msg("Set container's cpus limit")
	if err := c.cg.setCPULimit(cpus); err != nil {
		return err
	}
	return nil
}

// SetHostname sets container's hostname
// Default: ID[:12]
func (c *Container) SetHostname() {
	c.log.Info().Msg("Set hostname")
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
	c.cg.remove()
	return nil
}

// MountOverlayFS mounts filesystem for Container from an Image.
// It uses overlayFS for union mount of multiple layers.
func (c *Container) MountOverlayFS(img *images.Image) (filesystem.Unmounter, error) {
	c.log.Info().Str("image", img.Name).
		Msg("Mount filesystem for container from an image")
	if err := os.MkdirAll(c.RootFS, 0700); err != nil {
		return nil, errors.Wrapf(err, "can't create %s directory", c.RootFS)
	}

	imgLayers, err := img.Layers()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get image layers")
	}
	layers := make([]string, 0)
	for i := range imgLayers {
		digest, err := imgLayers[i].Digest()
		if err != nil {
			return nil, err
		}
		layers = append(layers, filepath.Join(constants.KokerImageLayersPath, digest.Hex))
	}
	unmounter, err := filesystem.OverlayMount(c.RootFS, layers, false)
	if err != nil {
		return unmounter, err
	}

	return unmounter, c.copyImageConfig(img)
}

func (c *Container) copyImageConfig(img *images.Image) error {
	c.log.Debug().Str("image", img.Name).Msg("Copy container config from image config")
	conCfg := filepath.Join(constants.KokerContainersPath, c.ID, "config.json")
	data, err := img.RawConfigFile()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(conCfg, data, 0655)
}

func (c *Container) LoadConfig() error {
	c.log.Info().Msg("Load container config from file")
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

// SetupNetwork configures network for the container
func (c *Container) SetupNetwork(bridge string) (filesystem.Unmounter, error) {
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
	defer unset()

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

func (c *Container) SetNetworkNamespace() (network.Unsetter, error) {
	netns := filepath.Join(constants.KokerNetNsPath, c.ID)
	return network.SetNetNSByFile(netns)
}
