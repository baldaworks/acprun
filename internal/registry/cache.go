package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultCacheDir returns the default user cache directory for ACP registry and agents.
// Follows $ACP_CACHE_DIR, os.UserCacheDir()/norma/acp-registry, or ~/.cache/norma/acp-registry.
func DefaultCacheDir() string {
	if envDir := os.Getenv("ACP_CACHE_DIR"); envDir != "" {
		return envDir
	}

	userCache, err := os.UserCacheDir()
	if err == nil && userCache != "" {
		return filepath.Join(userCache, "norma", "acp-registry")
	}

	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".cache", "norma", "acp-registry")
}

// CacheManager manages local caching of manifests, downloaded archives, and extracted binaries.
type CacheManager struct {
	rootDir string
}

// NewCacheManager creates a new CacheManager with root directory. If rootDir is empty, DefaultCacheDir() is used.
func NewCacheManager(rootDir string) *CacheManager {
	if rootDir == "" {
		rootDir = DefaultCacheDir()
	}
	return &CacheManager{rootDir: rootDir}
}

// RootDir returns the root cache directory path.
func (c *CacheManager) RootDir() string {
	return c.rootDir
}

// ManifestPath returns the path to the cached registry.json manifest file.
func (c *CacheManager) ManifestPath() string {
	return filepath.Join(c.rootDir, "manifest.json")
}

// AgentDir returns the cache directory path for a specific agent ID and version.
// e.g. <cache_dir>/<agent_id>/<version>
func (c *CacheManager) AgentDir(agentID, version string) string {
	return filepath.Join(c.rootDir, agentID, version)
}

// DownloadsDir returns the cache directory path for downloaded raw archives.
func (c *CacheManager) DownloadsDir() string {
	return filepath.Join(c.rootDir, "downloads")
}

// EnsureDir creates the directory with 0755 permissions if it does not exist.
func (c *CacheManager) EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// CleanOptions configures cache purge behavior.
type CleanOptions struct {
	All           bool
	ManifestsOnly bool
	DownloadsOnly bool
}

// Clean purges cache contents according to options.
func (c *CacheManager) Clean(opts CleanOptions) error {
	if opts.All {
		if err := os.RemoveAll(c.rootDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clean cache root %s: %w", c.rootDir, err)
		}
		return nil
	}

	if opts.ManifestsOnly {
		if err := os.Remove(c.ManifestPath()); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove manifest: %w", err)
		}
	}

	if opts.DownloadsOnly {
		if err := os.RemoveAll(c.DownloadsDir()); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove downloads: %w", err)
		}
	}

	return nil
}
