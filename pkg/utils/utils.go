package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/xid"
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

// GenUID returns a random string
func GenUID() string {
	return xid.New().String()
}

// CopyFile
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// Extract
func Extract(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()

	var tarReader *tar.Reader
	// Handle special case
	if strings.HasSuffix(tarball, "gz") {
		zipReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		tarReader = tar.NewReader(zipReader)
	} else {
		tarReader = tar.NewReader(reader)
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
