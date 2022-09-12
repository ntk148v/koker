package images

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

var imageRegistry registry

func init() {
	var err error
	imageRegistry, err = loadRegistry()
	if err != nil {
		log.Fatal().Err(err)
	}
}

// Manifest represents to manifest.json
type Manifest []struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// ParseManifest reads and unmarshals manifest.json to Manifest object
func ParseManifest(manifestPath string, m *Manifest) error {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, m); err != nil {
		return err
	}

	return nil
}

// DownloadImage gets image from repository and return image's SHA
func DownloadImage(src string) (string, error) {
	if s := strings.Split(src, ":"); len(s) == 1 {
		src = src + ":latest"
	}

	imageSHA, exist := imageRegistry.get(src)
	if !exist {
		log.Debug().Str("image", src).
			Msg("Get metadata for image")
		img, err := crane.Pull(src)
		if err != nil {
			return imageSHA, err
		}

		imageManifest, _ := img.Manifest()
		imageSHA = imageManifest.Config.Digest.Hex[:12]
		tmpPath := filepath.Join(constants.KokerTempPath, imageSHA)
		_ = os.Mkdir(tmpPath, 0755)
		tarball := filepath.Join(tmpPath, "package.tar")
		defer func() {
			// Cleanup - delete temp image files
			log.Debug().Msg("Delete temp image files")
			os.RemoveAll(tmpPath)
		}()

		// Save image as tar file
		log.Debug().Str("image", src).
			Msg("Save image as tar file")
		if err := crane.Save(img, imageSHA, tarball); err != nil {
			return imageSHA, err
		}

		// Untar tarball
		log.Debug().Str("tarball", tarball).
			Msg("Extract tar file")
		if err := utils.Extract(tarball, tmpPath); err != nil {
			return imageSHA, err
		}

		// Process layer tarballs
		log.Debug().Str("tarball", tarball).
			Msg("Process tarball's layers")
		manifestJson := filepath.Join(tmpPath, "manifest.json")
		configJson := filepath.Join(tmpPath, imageManifest.Config.Digest.String())

		m := Manifest{}
		ParseManifest(manifestJson, &m)
		if len(m) == 0 || len(m[0].Layers) == 0 {
			return imageSHA, errors.New("could not find any layers")
		} else if len(m) > 1 {
			return imageSHA, errors.New("unexpected mutiple manifestes")
		}

		imagePath := filepath.Join(constants.KokerImagesPath, imageSHA)
		_ = os.Mkdir(imagePath, 0755)
		// untar the layer files.
		for _, layer := range m[0].Layers {
			imageLayerDir := filepath.Join(imagePath, layer[:12], "fs")
			log.Debug().Str("tarball", tarball).
				Str("layerdir", imageLayerDir).
				Msg("Uncomressing layer to directory")
			_ = os.MkdirAll(imageLayerDir, 0755)
			if err := utils.Extract(filepath.Join(tmpPath, layer), imageLayerDir); err != nil {
				return imageSHA, err
			}
		}

		// copy manifest file for reference later
		if err := utils.CopyFile(manifestJson, filepath.Join(constants.KokerImagesPath,
			imageSHA, "manifest.json")); err != nil {
			return imageSHA, err
		}
		if err := utils.CopyFile(configJson, filepath.Join(constants.KokerImagesPath,
			imageSHA, imageSHA+".json")); err != nil {
			return imageSHA, err
		}

		// Store image metadata
		log.Debug().Msg("Store image metadata")
		imageRegistry.set(src, imageSHA)
		saveRegistry(imageRegistry)

		log.Debug().Str("image", src).
			Str("imgSHA", imageSHA).
			Msg("Download image successfully")
	} else {
		log.Debug().Str("image", src).
			Str("imgSHA", imageSHA).
			Msg("Image does exist, re-use")
	}
	return imageSHA, nil
}
