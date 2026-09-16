package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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

	type hardlink struct {
		oldname string
		newname string
	}
	var hardlinks []hardlink
	cleanTarget := filepath.Clean(target)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(cleanTarget, header.Name)
		cleanPath := filepath.Clean(path)
		if !strings.HasPrefix(cleanPath, cleanTarget+string(filepath.Separator)) && cleanPath != cleanTarget {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		info := header.FileInfo()

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(cleanPath, info.Mode()); err != nil {
				return err
			}
			continue
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return err
			}
		case tar.TypeLink:
			targetPath := filepath.Join(cleanTarget, header.Linkname)
			cleanTargetPath := filepath.Clean(targetPath)
			if !strings.HasPrefix(cleanTargetPath, cleanTarget+string(filepath.Separator)) && cleanTargetPath != cleanTarget {
				return fmt.Errorf("illegal link target in archive: %s", header.Linkname)
			}
			// Queue hard link creation after files are extracted
			hardlinks = append(hardlinks, hardlink{
				oldname: cleanTargetPath,
				newname: cleanPath,
			})
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, cleanPath); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
		}
	}

	for _, hl := range hardlinks {
		if err := os.MkdirAll(filepath.Dir(hl.newname), 0755); err != nil {
			return err
		}
		if err := os.Link(hl.oldname, hl.newname); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

// GenIPAddress generates ip address randomly in 172.69.0.0/16.
// It avoids the bridge gateway IP 172.69.0.1.
func GenIPAddress() string {
	octet3 := rand.Intn(256)
	var octet4 int
	if octet3 == 0 {
		// Avoid 0 (network) and 1 (bridge gateway)
		octet4 = rand.Intn(253) + 2
	} else if octet3 == 255 {
		// Avoid 255 (broadcast)
		octet4 = rand.Intn(254) + 1
	} else {
		octet4 = rand.Intn(254) + 1
	}
	return fmt.Sprintf("%s%d.%d/16", constants.KokerBridgeIPPrefix, octet3, octet4)
}

func CmdAndArgs(args []string) (command string, argv []string) {
	if len(args) == 0 {
		return
	}
	command = args[0]
	argv = args[1:]
	return
}

// GenTemplate inits and execute the given template
func GenTemplate(name, tempStr string, input any) error {
	temp := template.New(name)
	temp, err := temp.Parse(tempStr)
	if err != nil {
		return err
	}
	return temp.Execute(os.Stdout, input)
}
