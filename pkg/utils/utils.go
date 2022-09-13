package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
)

// CreateDir creates a directory if not exist
func CreateDir(dir string) error {
	_, err := os.Stat(dir)
	// If directory is not exist, create it
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			log.Error().Str("directory", dir).Err(err).
				Msg("Unable to create directory")
			return err
		}
	}
	return err
}

// InitKokerDirs creates all related directories
func InitKokerDirs() error {
	dirs := []string{
		constants.KokerHomePath, constants.KokerImagesPath,
		constants.KokerNetNsPath, constants.KokerContainersPath,
		constants.KokerTempPath,
	}

	for _, dir := range dirs {
		if err := CreateDir(dir); err != nil {
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

// Extract untars both .tar and .tar.gz files.
func Extract(reader io.Reader, target string, gz bool) error {
	var tarReader *tar.Reader

	if gz {
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

// GenMac generates MAC address
func GenMac() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	var mac net.HardwareAddr

	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}

	// Set the local bit
	buf[0] |= 2
	mac = append(mac, buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	return mac, err
}

// GenIPAddress generates ip address randomly (and dummy).
// NOTE(kiennt26): It doesn't check this IP
// address is used or not, as I assume there is just only 1 container
// run at time.
func GenIPAddress() string {
	// Hardcode
	return fmt.Sprintf("172.69.%d.%d", rand.Intn(254), rand.Intn(254))
}
