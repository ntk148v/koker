package images

import (
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

type Image struct {
	v1.Image
	ID       string
	Registry string
	Name     string
	Tag      string
	log      zerolog.Logger
}

// NewImage pulls image, constructs and returns a new Image
func NewImage(src string) (*Image, error) {
	log.Info().Str("image", src).Msg("Construct new Image instance")
	tag, err := name.NewTag(src)
	if err != nil {
		return nil, err
	}

	img, err := crane.Pull(tag.Name())
	if err != nil {
		return nil, err
	}

	imgCfgFile, _ := img.ConfigFile()

	return &Image{
		Image:    img,
		ID:       imgCfgFile.Config.Image,
		Registry: tag.RegistryStr(),
		Name:     tag.Name(),
		Tag:      tag.TagStr(),
		log:      log.With().Str("image", src).Logger(),
	}, nil
}

// Download downloads image's layers
func (i *Image) Download() error {
	i.log.Info().Str("registry", i.Registry).Msg("Download image from registry")
	layers, err := i.Layers()
	if err != nil {
		return err
	}

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return err
		}

		rc, err := layer.Uncompressed()
		if err != nil {
			return err
		}
		defer rc.Close()

		err = utils.Extract(rc, filepath.Join(constants.KokerImageLayersPath,
			digest.Hex), false)
		if err != nil {
			return err
		}
	}

	// Save to registry
	SetImage(i.ID, i.Name)
	return nil
}
