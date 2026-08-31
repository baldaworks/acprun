package resolver

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for name, content := range files {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		if strings.Contains(name, "bin") || strings.Contains(name, "agent") || strings.HasSuffix(name, "sh") {
			header.SetMode(0755)
		} else {
			header.SetMode(0644)
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry content: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func createTestTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		mode := int64(0644)
		if strings.HasSuffix(name, "bin") || strings.HasSuffix(name, "sh") {
			mode = 0755
		}
		header := &tar.Header{
			Name:     name,
			Mode:     mode,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write tar body: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func TestExtractZip(t *testing.T) {
	files := map[string]string{
		"bin/agent":     "#!/bin/sh\necho hello\n",
		"README.md":     "# Hello\n",
		"config/app.json": "{}",
	}

	zipPath := createTestZip(t, files)
	defer os.Remove(zipPath)

	destDir, err := os.MkdirTemp("", "extract-zip-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := ExtractArchive(zipPath, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	for relPath, expectedContent := range files {
		fullPath := filepath.Join(destDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("file %s not found: %v", relPath, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("file %s content mismatch: got %s, want %s", relPath, string(content), expectedContent)
		}
	}

	// Verify executable permission
	binPath := filepath.Join(destDir, "bin/agent")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("failed to stat executable: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("expected %s to have executable permissions, got mode %v", binPath, info.Mode())
	}
}

func TestExtractTarGz(t *testing.T) {
	files := map[string]string{
		"agent-bin": "#!/bin/sh\necho tar\n",
		"doc.txt":   "Tar documentation",
	}

	tarPath := createTestTarGz(t, files)
	defer os.Remove(tarPath)

	destDir, err := os.MkdirTemp("", "extract-tar-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := ExtractArchive(tarPath, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	for relPath, expectedContent := range files {
		fullPath := filepath.Join(destDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("file %s not found: %v", relPath, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("file %s content mismatch: got %s, want %s", relPath, string(content), expectedContent)
		}
	}
}

func TestZipSlipProtection(t *testing.T) {
	maliciousFiles := map[string]string{
		"../../evil.txt": "pwned",
	}

	zipPath := createTestZip(t, maliciousFiles)
	defer os.Remove(zipPath)

	destDir, err := os.MkdirTemp("", "zipslip-test-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	err = ExtractArchive(zipPath, destDir)
	if err == nil {
		t.Errorf("expected error on Zip Slip archive, but got nil")
	} else if !strings.Contains(err.Error(), "Zip Slip") {
		t.Errorf("expected Zip Slip error, got: %v", err)
	}
}
