package resolver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/baldaworks/acprun/internal/registry"
)

// ResolvedCommand represents the fully resolved executable specification for an ACP agent.
type ResolvedCommand struct {
	AgentID       string            `json:"agent_id"`
	Version       string            `json:"version"`
	Format        string            `json:"format"` // "binary", "npx", "uvx"
	Executable    string            `json:"executable"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	ExtractedPath string            `json:"extracted_path,omitempty"`
}

// ResolveOptions configures the resolution process.
type ResolveOptions struct {
	Platform   string            // Target platform (defaults to host platform)
	Format     string            // Preferred format ("binary", "npx", "uvx", or "" for auto)
	NoDownload bool              // Do not attempt to download binary archives
	ExtraArgs  []string          // Extra CLI arguments to append
	ExtraEnv   map[string]string // Extra environment variables to merge
}

// Resolver resolves agents into runnable command vectors.
type Resolver struct {
	cacheManager *registry.CacheManager
	httpClient   *http.Client
}

// NewResolver creates a new agent Resolver.
func NewResolver(cacheManager *registry.CacheManager, httpClient *http.Client) *Resolver {
	if cacheManager == nil {
		cacheManager = registry.NewCacheManager("")
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Minute,
		}
	}
	return &Resolver{
		cacheManager: cacheManager,
		httpClient:   httpClient,
	}
}

// Resolve resolves an agent into an executable ResolvedCommand according to options.
func (r *Resolver) Resolve(ctx context.Context, agent *registry.Agent, opts ResolveOptions) (*ResolvedCommand, error) {
	if agent == nil {
		return nil, fmt.Errorf("cannot resolve nil agent")
	}

	targetPlatform := opts.Platform
	if targetPlatform == "" {
		detected, err := DetectCurrentPlatform()
		if err != nil {
			return nil, fmt.Errorf("failed to detect current platform: %w", err)
		}
		targetPlatform = detected
	} else {
		normalized, err := NormalizePlatform(targetPlatform)
		if err != nil {
			return nil, err
		}
		targetPlatform = normalized
	}

	var cmd *ResolvedCommand
	var err error

	format := opts.Format
	if format != "" {
		switch format {
		case "binary":
			cmd, err = r.resolveBinary(ctx, agent, targetPlatform, opts.NoDownload)
		case "npx":
			cmd, err = r.resolveNPX(agent)
		case "uvx":
			cmd, err = r.resolveUVX(agent)
		default:
			return nil, fmt.Errorf("unknown distribution format requested: %q", format)
		}
	} else {
		// Automatic distribution selection: binary (if supported) -> npx -> uvx
		if _, bErr := agent.GetBinaryTarget(targetPlatform); bErr == nil {
			cmd, err = r.resolveBinary(ctx, agent, targetPlatform, opts.NoDownload)
		} else if agent.Distribution.NPX != nil {
			cmd, err = r.resolveNPX(agent)
		} else if agent.Distribution.UVX != nil {
			cmd, err = r.resolveUVX(agent)
		} else {
			return nil, fmt.Errorf("agent %q does not provide a distribution for platform %q", agent.ID, targetPlatform)
		}
	}

	if err != nil {
		return nil, err
	}

	// Append extra arguments
	if len(opts.ExtraArgs) > 0 {
		cmd.Args = append(cmd.Args, opts.ExtraArgs...)
	}

	// Merge extra environment variables
	if len(opts.ExtraEnv) > 0 {
		if cmd.Env == nil {
			cmd.Env = make(map[string]string)
		}
		for k, v := range opts.ExtraEnv {
			cmd.Env[k] = v
		}
	}

	return cmd, nil
}
