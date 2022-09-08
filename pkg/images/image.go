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

type manifest []struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}
type imageConfigDetails struct {
	Env []string `json:"Env"`
	Cmd []string `json:"Cmd"`
}
type imageConfig struct {
	Config imageConfigDetails `json:"config"`
}

func parseManifest(manifestPath string, m *manifest) error {
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
		log.Debug().Msgf("Get metadata for %s", src)
		img, err := crane.Pull(src)
		if err != nil {
			return imageSHA, err
		}

		imageManifest, _ := img.Manifest()
		imageSHA = imageManifest.Config.Digest.Hex[:12]
		tmpPath := filepath.Join(utils.KokerTempPath, imageSHA)
		_ = os.Mkdir(tmpPath, 0755)
		tarball := filepath.Join(tmpPath, "package.tar")
		defer func() {
			// Cleanup - delete temp image files
			log.Debug().Msg("Delete temp image files")
			os.RemoveAll(tmpPath)
		}()

		// Save image as tar file
		log.Debug().Msgf("Save image %s as tar file", src)
		if err := crane.Save(img, imageSHA, tarball); err != nil {
			return imageSHA, err
		}

		// Untar tarball
		log.Debug().Msgf("Extract tar file %s", tarball)
		if err := utils.Extract(tarball, tmpPath); err != nil {
			return imageSHA, err
		}

		// Process layer tarballs
		log.Debug().Msg("Process layer tarballs")
		manifestJson := filepath.Join(tmpPath, "manifest.json")
		configJson := filepath.Join(tmpPath, imageManifest.Config.Digest.Hex+".json")

		m := manifest{}
		parseManifest(manifestJson, &m)
		if len(m) == 0 || len(m[0].Layers) == 0 {
			return imageSHA, errors.New("could not find any layers")
		} else if len(m) > 1 {
			return imageSHA, errors.New("unexpected mutiple manifestes")
		}

		imagePath := filepath.Join(utils.KokerImagesPath, imageSHA)
		_ = os.Mkdir(imagePath, 0755)
		// untar the layer files.
		for _, layer := range m[0].Layers {
			imageLayerDir := filepath.Join(imagePath, layer[:12], "fs")
			log.Debug().Msgf("Uncomressing layer to %s", imageLayerDir)
			_ = os.MkdirAll(imageLayerDir, 0755)
			if err := utils.Extract(filepath.Join(tmpPath, layer), imageLayerDir); err != nil {
				return imageSHA, err
			}
		}

		// copy manifest file for reference later
		utils.CopyFile(manifestJson, filepath.Join(utils.KokerImagesPath,
			imageSHA, "manifest.json"))
		utils.CopyFile(configJson, filepath.Join(utils.KokerImagesPath,
			imageSHA, imageSHA+".json"))

		// Store image metadata
		log.Debug().Msg("Store image metadata")
		imageRegistry.set(src, imageSHA)
		saveRegistry(imageRegistry)
	}
	return imageSHA, nil
}
