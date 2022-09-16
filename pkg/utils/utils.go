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
	"syscall"
	"text/template"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"

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

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		case tar.TypeReg:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(file, tarReader)
			if err != nil {
				return err
			}
		case tar.TypeLink:
			link := filepath.Join(target, header.Name)
			linkTarget := filepath.Join(target, header.Linkname)
			// lazy link creation. just to make sure all files are available
			defer os.Link(link, linkTarget)
		case tar.TypeSymlink:
			linkPath := filepath.Join(target, header.Name)
			if err := os.Symlink(header.Linkname, linkPath); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
		}
	}
	return nil
}

// GenIPAddress generates ip address randomly (and dummy).
// NOTE(kiennt26): It doesn't check this IP
// address is used or not, as I assume there is just only 1 container
// run at time.
func GenIPAddress() string {
	// Hardcode
	return fmt.Sprintf("172.69.%d.%d/16", rand.Intn(254), rand.Intn(254))
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

// SetNamespace calls setns syscall for set of flags. It changes
// current process namespace to namespace of another process which
// can be specified by pid.
//
// NOTE: A process may not be reassociated with a new mount namespace
// if it is multi-threaded. Changing the mount namespace requires that
// the caller possess both CAP_SYS_CHROOT and CAP_SYS_ADMIN capabilities
// in its own user namespace and CAP_SYS_ADMIN in the target mount namespace.
func SetNamespace(pid string, flag int) error {
	nsBase := filepath.Join("/proc", pid, "ns")
	ns := map[int]string{
		syscall.CLONE_NEWIPC: "ipc",
		syscall.CLONE_NEWNS:  "mnt",
		syscall.CLONE_NEWNET: "net",
		syscall.CLONE_NEWPID: "pid",
		syscall.CLONE_NEWUTS: "uts",
	}

	for k, v := range ns {
		if flag&k == 0 {
			continue
		}
		nsFile, err := os.Open(filepath.Join(nsBase, v))
		if err != nil {
			return errors.Wrapf(err, "can't open %s", nsFile)
		}

		if err := unix.Setns(int(nsFile.Fd()), k); err != nil {
			return errors.Wrapf(err, "can't setns to %s", v)
		}
	}

	return nil
}
