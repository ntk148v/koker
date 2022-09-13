package containers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/filesystem"
	"github.com/ntk148v/koker/pkg/images"
	"github.com/ntk148v/koker/pkg/utils"
)

type Container struct {
	Config    *v1.Config
	ID        string
	RootFS    string
	Pids      []int
	log       zerolog.Logger
	unmounter filesystem.Unmounter
	cg        *cgroups
	mem       int
	swap      int
	pids      int
	cpus      float64
}

// NewContainer returns a new Container instance with random digest
func NewContainer() *Container {
	id := utils.GenUID()
	return &Container{
		Config: new(v1.Config),
		ID:     id,
		log:    log.With().Str("container", id).Logger(),
		cg:     newCGroup(filepath.Join(constants.KokerApp, id)),
	}
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

func (c *Container) Delete() error {
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
	imgSrc := img.Name + ":" + img.Tag
	c.log.Info().Str("image", imgSrc).
		Msg("Mount filesystem for container from an image")
	target := filepath.Join(constants.KokerContainersPath, c.ID, "mnt")
	if err := os.MkdirAll(target, 0700); err != nil {
		return nil, errors.Wrapf(err, "can't create %s directory", target)
	}

	c.RootFS = target
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
	unmounter, err := filesystem.OverlayMount(target, layers, false)
	if err != nil {
		return unmounter, err
	}

	return unmounter, c.loadConfig(img)
}

func (c *Container) loadConfig(img *images.Image) error {
	imgSrc := img.Name + ":" + img.Tag
	c.log.Info().Str("image", imgSrc).Msg("Load container config from image config")
	c.log.Debug().Str("image", imgSrc).Msg("Copy container config from image config")
	conCfg := filepath.Join(constants.KokerContainersPath, c.ID, "config.json")

	// Load config
	imgCfg, err := img.ConfigFile()
	if err != nil {
		return errors.Wrap(err, "unable to load image's config file")
	}
	c.Config = imgCfg.Config.DeepCopy()

	// Save to file
	raw, err := img.RawConfigFile()
	if err != nil {
		return errors.Wrap(err, "unable to get image's raw config")
	}
	return ioutil.WriteFile(conCfg, raw, 0655)
}
