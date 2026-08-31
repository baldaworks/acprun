package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultRegistryURL is the official ACP CDN registry URL.
const DefaultRegistryURL = "https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json"

// DefaultCacheTTL is the default time-to-live for cached registry manifests.
const DefaultCacheTTL = 1 * time.Hour

// ClientOptions configures the Registry Client.
type ClientOptions struct {
	RegistryURL string
	CacheDir    string
	CacheTTL    time.Duration
	Offline     bool
	HTTPClient  *http.Client
}

// Client interacts with ACP registries to fetch and cache agent manifests.
type Client struct {
	registryURL  string
	cacheManager *CacheManager
	cacheTTL     time.Duration
	offline      bool
	httpClient   *http.Client
	mu           sync.Mutex
	cached       *Registry
}

// NewClient creates a new Registry Client with options.
func NewClient(opts ClientOptions) *Client {
	url := opts.RegistryURL
	if url == "" {
		if envURL := os.Getenv("ACP_REGISTRY_URL"); envURL != "" {
			url = envURL
		} else {
			url = DefaultRegistryURL
		}
	}

	ttl := opts.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		registryURL:  url,
		cacheManager: NewCacheManager(opts.CacheDir),
		cacheTTL:     ttl,
		offline:      opts.Offline,
		httpClient:   httpClient,
	}
}

// CacheManager returns the underlying CacheManager.
func (c *Client) CacheManager() *CacheManager {
	return c.cacheManager
}

// FetchRegistry retrieves the ACP registry manifest, respecting caching and offline options.
func (c *Client) FetchRegistry(ctx context.Context) (*Registry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil && !c.offline {
		return c.cached, nil
	}

	manifestPath := c.cacheManager.ManifestPath()
	slog.Debug("checking registry manifest cache", "manifest_path", manifestPath, "offline", c.offline)

	// In offline mode, strictly load from cache
	if c.offline {
		reg, err := c.loadFromCache(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("offline mode requested but cached registry manifest is unavailable: %w", err)
		}
		c.cached = reg
		slog.Debug("loaded registry manifest in offline mode", "manifest_path", manifestPath, "agents_count", len(reg.Agents))
		return reg, nil
	}

	// Check if cached manifest exists and is fresh
	if info, err := os.Stat(manifestPath); err == nil {
		if time.Since(info.ModTime()) < c.cacheTTL {
			reg, err := c.loadFromCache(manifestPath)
			if err == nil {
				c.cached = reg
				slog.Debug("loaded fresh registry manifest from cache", "manifest_path", manifestPath, "agents_count", len(reg.Agents))
				return reg, nil
			}
		}
	}

	// Fetch from network
	slog.Debug("fetching registry manifest from network", "url", c.registryURL)
	reg, fetchErr := c.fetchFromNetwork(ctx)
	if fetchErr == nil {
		// Save to cache atomically
		_ = c.saveToCache(manifestPath, reg)
		c.cached = reg
		slog.Debug("fetched and cached registry manifest", "url", c.registryURL, "agents_count", len(reg.Agents))
		return reg, nil
	}

	// If network fetch fails, fallback to existing cached manifest (even if stale)
	cachedReg, cacheErr := c.loadFromCache(manifestPath)
	if cacheErr == nil {
		c.cached = cachedReg
		slog.Warn("network fetch failed, falling back to cached manifest", "url", c.registryURL, "error", fetchErr)
		return cachedReg, nil
	}

	return nil, fmt.Errorf("failed to fetch ACP registry from %s: %w", c.registryURL, fetchErr)
}

func (c *Client) fetchFromNetwork(ctx context.Context) (*Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "acprun/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(body, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry JSON: %w", err)
	}

	if reg.Version == "" {
		return nil, errors.New("invalid registry JSON: missing 'version' field")
	}

	return &reg, nil
}

func (c *Client) loadFromCache(manifestPath string) (*Registry, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("corrupted manifest cache: %w", err)
	}
	return &reg, nil
}

func (c *Client) saveToCache(manifestPath string, reg *Registry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(manifestPath)
	if err := c.cacheManager.EnsureDir(dir); err != nil {
		return err
	}

	tmpFile := manifestPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, manifestPath)
}

// GetAgent retrieves a specific agent by ID.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	reg, err := c.FetchRegistry(ctx)
	if err != nil {
		return nil, err
	}

	for _, a := range reg.Agents {
		if a.ID == agentID {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("agent %q not found in registry", agentID)
}

// ListAgents returns all agents in the registry.
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	reg, err := c.FetchRegistry(ctx)
	if err != nil {
		return nil, err
	}
	return reg.Agents, nil
}
