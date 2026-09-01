package resolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/baldaworks/acprun/internal/registry"
)

func (r *Resolver) resolveBinary(ctx context.Context, agent *registry.Agent, targetPlatform string, noDownload bool) (*ResolvedCommand, error) {
	target, err := agent.GetBinaryTarget(targetPlatform)
	if err != nil {
		return nil, err
	}

	agentDir := r.cacheManager.AgentDir(agent.ID, agent.Version)
	cleanCmd := normalizeCmdPath(target.Cmd)
	execPath := filepath.Join(agentDir, cleanCmd)

	// Check if already extracted and executable exists
	if info, err := os.Stat(execPath); err == nil && !info.IsDir() {
		_ = os.Chmod(execPath, 0755)
		slog.Debug("using cached agent binary", "agent_id", agent.ID, "executable", execPath)
		return &ResolvedCommand{
			AgentID:       agent.ID,
			Version:       agent.Version,
			Format:        "binary",
			Executable:    execPath,
			Args:          target.Args,
			Env:           target.Env,
			WorkingDir:    agentDir,
			ExtractedPath: agentDir,
		}, nil
	}

	if noDownload {
		return nil, fmt.Errorf("binary for agent %q (%s) is not cached locally and --no-download was requested", agent.ID, agent.Version)
	}

	// Download archive
	downloadsDir := r.cacheManager.DownloadsDir()
	if err := r.cacheManager.EnsureDir(downloadsDir); err != nil {
		return nil, fmt.Errorf("failed to create downloads directory: %w", err)
	}

	archiveFilename := filepath.Base(target.Archive)
	if archiveFilename == "" || archiveFilename == "." {
		archiveFilename = fmt.Sprintf("%s-%s-%s.archive", agent.ID, agent.Version, targetPlatform)
	}
	archivePath := filepath.Join(downloadsDir, archiveFilename)

	if verifyFileSHA256(archivePath, target.SHA256) {
		slog.Debug("using cached download archive", "agent_id", agent.ID, "archive", archivePath)
	} else {
		slog.Info("downloading agent binary archive", "agent_id", agent.ID, "version", agent.Version, "url", target.Archive)
		if err := r.downloadAndVerify(ctx, target.Archive, archivePath, target.SHA256); err != nil {
			return nil, err
		}
	}

	// Extract archive to agentDir
	if err := r.cacheManager.EnsureDir(agentDir); err != nil {
		return nil, fmt.Errorf("failed to create agent directory: %w", err)
	}

	slog.Debug("extracting agent binary archive", "agent_id", agent.ID, "archive", archivePath, "dest", agentDir)
	if err := ExtractArchive(archivePath, agentDir); err != nil {
		// Clean incomplete extraction directory on error
		_ = os.RemoveAll(agentDir)
		return nil, fmt.Errorf("failed to extract archive for %s: %w", agent.ID, err)
	}

	// Verify the executable exists
	if info, err := os.Stat(execPath); err != nil || info.IsDir() {
		return nil, fmt.Errorf("executable %q not found in extracted archive for %s (looked at %s)", target.Cmd, agent.ID, execPath)
	}

	// Set executable permissions
	_ = os.Chmod(execPath, 0755)
	slog.Debug("extracted agent binary successfully", "agent_id", agent.ID, "executable", execPath)

	return &ResolvedCommand{
		AgentID:       agent.ID,
		Version:       agent.Version,
		Format:        "binary",
		Executable:    execPath,
		Args:          target.Args,
		Env:           target.Env,
		WorkingDir:    agentDir,
		ExtractedPath: agentDir,
	}, nil
}

func normalizeCmdPath(cmd string) string {
	cmd = strings.TrimPrefix(cmd, "./")
	cmd = strings.TrimPrefix(cmd, ".\\")
	// Convert any backslashes to system separator
	cmd = filepath.FromSlash(strings.ReplaceAll(cmd, "\\", "/"))
	return cmd
}

func (r *Resolver) downloadAndVerify(ctx context.Context, url, destPath, expectedSHA256 string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "acprun/1.0")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download archive from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download archive from %s: HTTP %d %s", url, resp.StatusCode, resp.Status)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), "download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary download file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write downloaded archive: %w", err)
	}
	tmpFile.Close()

	if expectedSHA256 != "" {
		actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(actualSHA256, expectedSHA256) {
			return fmt.Errorf("SHA256 checksum mismatch for %s: expected %s, got %s", url, expectedSHA256, actualSHA256)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to finalize downloaded archive: %w", err)
	}

	return nil
}

func verifyFileSHA256(filePath, expectedSHA256 string) bool {
	if expectedSHA256 == "" {
		return false
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	return strings.EqualFold(actual, expectedSHA256)
}

