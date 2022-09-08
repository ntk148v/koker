package utils

import (
	"os"

	"github.com/rs/zerolog/log"
)

const (
	KokerHomePath       = "/var/lib/koker"
	KokerTempPath       = KokerHomePath + "/tmp"
	KokerImagesPath     = KokerHomePath + "/images"
	KokerContainersPath = KokerHomePath + "/containers"
	KokerNetNsPath      = KokerHomePath + "/netns"
)

// createDir creates a directory if not exist
func createDir(dir string) error {
	_, err := os.Stat(dir)
	// If directory is not exist, create it
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			log.Error().Err(err).Msgf("error creating directory %s", dir)
			return err
		}
	}
	return err
}

// InitKokerDirs creates all related directories
func InitKokerDirs() error {
	for _, dir := range []string{KokerHomePath, KokerImagesPath, KokerNetNsPath, KokerContainersPath, KokerTempPath} {
		if err := createDir(dir); err != nil {
			return err
		}
	}
	return nil
}
