package images

import (
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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
}

// NewImage pulls image, constructs and returns a new Image
func NewImage(src string) (*Image, error) {
	log.Info().Str("src", src).Msg("Construct new Image instace")
	tag, err := name.NewTag(src)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("src", src).Msg("Pull image")
	img, err := crane.Pull(tag.Name())
	if err != nil {
		return nil, err
	}

	digest, err := img.Digest()
	if err != nil {
		return nil, err
	}

	return &Image{
		Image:    img,
		ID:       digest.Hex,
		Registry: tag.RegistryStr(),
		Name:     tag.Name(),
		Tag:      tag.TagStr(),
	}, nil
}

// Download downloads image's layers
func (i *Image) Download() error {
	log.Info().Str("name", i.Name).Str("tag", i.Tag).Msg("Download image's layers")
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
	SetImage(i.ID, *i)
	return nil
}
