package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdAndArgs(t *testing.T) {
	cmd, argv := CmdAndArgs([]string{"echo", "hello", "world"})
	if cmd != "echo" {
		t.Errorf("expected echo, got %s", cmd)
	}
	if len(argv) != 2 || argv[0] != "hello" || argv[1] != "world" {
		t.Errorf("unexpected argv: %v", argv)
	}

	emptyCmd, emptyArgv := CmdAndArgs([]string{})
	if emptyCmd != "" || len(emptyArgv) != 0 {
		t.Errorf("expected empty cmd and argv, got %s and %v", emptyCmd, emptyArgv)
	}
}

func TestGenUID(t *testing.T) {
	uid1 := GenUID()
	uid2 := GenUID()
	if uid1 == "" || uid2 == "" {
		t.Errorf("generated UID should not be empty")
	}
	if uid1 == uid2 {
		t.Errorf("consecutive UIDs should be unique")
	}
}

func TestGenIPAddress(t *testing.T) {
	for i := 0; i < 100; i++ {
		ipStr := GenIPAddress()
		ip, _, err := net.ParseCIDR(ipStr)
		if err != nil {
			t.Fatalf("invalid CIDR generated: %s: %v", ipStr, err)
		}
		if ip.String() == "172.69.0.1" {
			t.Errorf("generated IP must not be the gateway IP 172.69.0.1")
		}
		if !strings.HasPrefix(ipStr, "172.69.") {
			t.Errorf("expected IP in 172.69.0.0/16, got %s", ipStr)
		}
	}
}

func TestCreateDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "sub", "dir")
	if err := CreateDir(newDir); err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}
	fi, err := os.Stat(newDir)
	if err != nil || !fi.IsDir() {
		t.Fatalf("directory was not created properly: %v", err)
	}
	// Calling again should be idempotent
	if err := CreateDir(newDir); err != nil {
		t.Fatalf("subsequent CreateDir failed: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	content := []byte("hello koker")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read dst: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("content mismatch: expected %s, got %s", content, dstContent)
	}
}

func TestGenTemplate(t *testing.T) {
	tmpl := `Hello {{ .Name }}!`
	data := struct{ Name string }{Name: "Koker"}
	if err := GenTemplate("test", tmpl, data); err != nil {
		t.Fatalf("GenTemplate failed: %v", err)
	}
}

func TestExtract(t *testing.T) {
	tmpDir := t.TempDir()
	tarballPath := filepath.Join(tmpDir, "test.tar.gz")

	// Create a test tar.gz with directory, file, symlink, and hard link
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Directory
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}); err != nil {
		t.Fatal(err)
	}

	// Regular file
	fileContent := []byte("koker regular file")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/hello.txt",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(fileContent)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(fileContent); err != nil {
		t.Fatal(err)
	}

	// Hard link
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/hello_link.txt",
		Linkname: "subdir/hello.txt",
		Typeflag: tar.TypeLink,
	}); err != nil {
		t.Fatal(err)
	}

	// Symlink
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/hello_symlink.txt",
		Linkname: "hello.txt",
		Typeflag: tar.TypeSymlink,
	}); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()

	if err := os.WriteFile(tarballPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := Extract(tarballPath, extractDir); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify extracted files
	read, err := os.ReadFile(filepath.Join(extractDir, "subdir", "hello.txt"))
	if err != nil || string(read) != string(fileContent) {
		t.Fatalf("regular file extraction error: %v, content: %s", err, string(read))
	}

	readLink, err := os.ReadFile(filepath.Join(extractDir, "subdir", "hello_link.txt"))
	if err != nil || string(readLink) != string(fileContent) {
		t.Fatalf("hardlink extraction error: %v, content: %s", err, string(readLink))
	}

	readSymlink, err := os.ReadFile(filepath.Join(extractDir, "subdir", "hello_symlink.txt"))
	if err != nil || string(readSymlink) != string(fileContent) {
		t.Fatalf("symlink extraction error: %v, content: %s", err, string(readSymlink))
	}
}

func TestExtractZipSlip(t *testing.T) {
	tmpDir := t.TempDir()
	tarballPath := filepath.Join(tmpDir, "slip.tar")

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Attempt directory traversal
	if err := tw.WriteHeader(&tar.Header{
		Name:     "../../etc/passwd",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     4,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte("evil"))
	tw.Close()

	if err := os.WriteFile(tarballPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	extractDir := filepath.Join(tmpDir, "safe")
	err := Extract(tarballPath, extractDir)
	if err == nil {
		t.Errorf("Extract should reject path traversal with error")
	}
}
