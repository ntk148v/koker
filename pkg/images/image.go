package images

import (
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

type Image struct {
	Metadata Metadata
	log      zerolog.Logger
}

type Metadata struct {
	ID         string       `json:"id"`
	Digest     string       `json:"digest"`
	Manifest   *v1.Manifest `json:"manifest"`
	Registry   string       `json:"registry"`
	Repository string       `json:"repository"`
	Name       string       `json:"name"`
	Tag        string       `json:"tag"`
}

// NewImage pulls image, constructs and returns a new Image
func NewImage(src string) (*Image, error) {
	log.Info().Str("image", src).Msg("Construct new Image instance")
	tag, err := name.NewTag(src)
	if err != nil {
		return nil, err
	}

	var img Image
	img.log = log.With().Str("image", tag.Name()).Logger()

	// Check image exist
	metadata, exist := GetImage(tag.Name())
	if !exist {
		if err := img.Download(tag); err != nil {
			return nil, errors.Wrap(err, "unable to pull image")
		}
	} else {
		img.log.Info().Msg("Image exists, reuse")
		img.Metadata = metadata
	}

	return &img, nil
}

// Download downloads and extract image's layers
func (i *Image) Download(tag name.Tag) error {
	i.log.Info().Msg("Download image from registry")
	i.log.Debug().Msg("Pull image's metadata")
	img, err := crane.Pull(tag.Name())
	if err != nil {
		return errors.Wrap(err, "unable to pull image metadata")
	}

	// Get manifest
	manifest, err := img.Manifest()
	if err != nil {
		return err
	}

	// Get image's id
	imgID, err := img.ConfigName()
	if err != nil {
		return errors.Wrap(err, "unable to get image config")
	}

	// Store image metadata
	imgSHA := manifest.Config.Digest.Hex
	i.Metadata = Metadata{
		ID:         imgID.Hex,
		Digest:     imgSHA,
		Tag:        tag.TagStr(),
		Repository: tag.RepositoryStr(),
		Name:       tag.Name(),
		Registry:   tag.RegistryStr(),
		Manifest:   manifest,
	}

	// Create temp
	tmpPath := filepath.Join(constants.KokerTempPath, imgSHA)
	_ = utils.CreateDir(tmpPath)
	tarball := filepath.Join(tmpPath, "package.tar")
	defer func() {
		// Cleanup
		i.log.Debug().Msg("Delete temp image files")
		os.RemoveAll(tmpPath)
	}()

	// Save image as tar file
	i.log.Debug().Str("tarball", tarball).Msg("Save image as tar file")
	if err := crane.Save(img, imgSHA, tarball); err != nil {
		return errors.Wrap(err, "unable to save image as tar file")
	}

	// Untar tarball
	i.log.Debug().Str("tarball", tarball).Msg("Extract tar file")
	if err := utils.Extract(tarball, tmpPath); err != nil {
		return err
	}

	imgPath := filepath.Join(constants.KokerImagesPath, imgSHA)
	_ = utils.CreateDir(imgPath)

	// Dump image config file
	configPath := filepath.Join(imgPath, "config.json")
	data, err := img.RawConfigFile()
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0655); err != nil {
		return err
	}

	// Process tarball's layers
	log.Debug().Str("tarball", tarball).
		Msg("Process tarball's layers")
	// untar the layer files
	for _, layer := range manifest.Layers {
		imgLayerPath := filepath.Join(imgPath, layer.Digest.Hex)
		log.Debug().Str("tarball", tarball).
			Str("layerdir", imgLayerPath).
			Msg("Extract layer to directory")
		_ = utils.CreateDir(imgLayerPath)
		if err := utils.Extract(filepath.Join(tmpPath, layer.Digest.Hex+".tar.gz"), imgLayerPath); err != nil {
			return errors.Wrap(err, "unable to extract tarball's layer")
		}
	}

	SetImage(tag.Name(), i.Metadata)
	return nil
}
