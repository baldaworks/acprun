package resolver

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive extracts an archive (.zip, .tar.gz, .tgz, .tar.bz2, .tar) to destDir with Zip Slip protection.
func ExtractArchive(archivePath, destDir string) error {
	destDir = filepath.Clean(destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2"):
		return extractTarBz2(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar"):
		return extractTar(archivePath, destDir)
	default:
		// Attempt to inspect magic bytes if extension is missing/unknown
		return extractByInspection(archivePath, destDir)
	}
}

func isSafePath(destDir, targetPath string) bool {
	destClean := filepath.Clean(destDir)
	targetClean := filepath.Clean(targetPath)
	if targetClean == destClean {
		return true
	}
	return strings.HasPrefix(targetClean, destClean+string(filepath.Separator))
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file %s: %w", archivePath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		if !isSafePath(destDir, targetPath) {
			return fmt.Errorf("illegal file path in zip archive (Zip Slip attempt): %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file inside zip %s: %w", f.Name, err)
		}

		mode := f.Mode()
		if mode&0111 != 0 {
			mode = 0755
		} else {
			mode = 0644
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file %s: %w", targetPath, err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		// Ensure permissions are set explicitly
		_ = os.Chmod(targetPath, mode)
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader for %s: %w", archivePath, err)
	}
	defer gzReader.Close()

	return extractTarReader(tar.NewReader(gzReader), destDir)
}

func extractTarBz2(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}
	defer file.Close()

	bzReader := bzip2.NewReader(file)
	return extractTarReader(tar.NewReader(bzReader), destDir)
}

func extractTar(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}
	defer file.Close()

	return extractTarReader(tar.NewReader(file), destDir)
}

func extractTarReader(tr *tar.Reader, destDir string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar stream: %w", err)
		}

		targetPath := filepath.Join(destDir, header.Name)
		if !isSafePath(destDir, targetPath) {
			return fmt.Errorf("illegal file path in tar archive (Zip Slip attempt): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			mode := os.FileMode(header.Mode)
			if mode&0111 != 0 {
				mode = 0755
			} else {
				mode = 0644
			}

			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()
			_ = os.Chmod(targetPath, mode)

		case tar.TypeSymlink:
			// Resolve symlink target to verify it stays within extraction root
			linkTarget := header.Linkname
			var absLinkTarget string
			if filepath.IsAbs(linkTarget) {
				absLinkTarget = filepath.Clean(linkTarget)
			} else {
				absLinkTarget = filepath.Clean(filepath.Join(filepath.Dir(targetPath), linkTarget))
			}
			if !isSafePath(destDir, absLinkTarget) {
				return fmt.Errorf("illegal symlink target in archive (Zip Slip attempt): %s -> %s", header.Name, header.Linkname)
			}
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}
		}
	}
	return nil
}

func extractByInspection(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", archivePath, err)
	}
	defer file.Close()

	var buf [4]byte
	n, err := file.Read(buf[:])
	if err != nil || n < 2 {
		return fmt.Errorf("unsupported or empty archive: %s", archivePath)
	}

	// Zip magic number: PK\x03\x04
	if buf[0] == 'P' && buf[1] == 'K' {
		return extractZip(archivePath, destDir)
	}
	// Gzip magic number: \x1f\x8b
	if buf[0] == 0x1f && buf[1] == 0x8b {
		return extractTarGz(archivePath, destDir)
	}
	// Bzip2 magic number: BZ
	if buf[0] == 'B' && buf[1] == 'Z' {
		return extractTarBz2(archivePath, destDir)
	}

	return fmt.Errorf("unknown archive format for %s", archivePath)
}
