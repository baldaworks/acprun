// Package registry provides data models and client operations for ACP registries.
package registry

import (
	"fmt"
	"strings"
)

// Registry represents an ACP registry manifest (registry.json).
type Registry struct {
	Version string  `json:"version"`
	Agents  []Agent `json:"agents"`
}

// Agent represents an ACP agent definition in the registry.
type Agent struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description,omitempty"`
	Repository   string       `json:"repository,omitempty"`
	Website      string       `json:"website,omitempty"`
	Authors      []string     `json:"authors,omitempty"`
	License      string       `json:"license,omitempty"`
	Icon         string       `json:"icon,omitempty"`
	Distribution Distribution `json:"distribution"`
}

// Distribution represents the distribution methods available for an agent.
type Distribution struct {
	Binary map[string]BinaryTarget `json:"binary,omitempty"`
	NPX    *NPXTarget              `json:"npx,omitempty"`
	UVX    *UVXTarget              `json:"uvx,omitempty"`
}

// BinaryTarget defines the binary download and execution configuration for a platform.
type BinaryTarget struct {
	Archive string            `json:"archive"`
	Cmd     string            `json:"cmd"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	SHA256  string            `json:"sha256,omitempty"`
}

// NPXTarget defines the NPX package and execution configuration.
type NPXTarget struct {
	Package string            `json:"package"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// UVXTarget defines the UVX package and execution configuration.
type UVXTarget struct {
	Package string            `json:"package"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// DistributionTypes returns the distribution formats provided by the agent.
func (a *Agent) DistributionTypes() []string {
	var types []string
	if len(a.Distribution.Binary) > 0 {
		types = append(types, "binary")
	}
	if a.Distribution.NPX != nil {
		types = append(types, "npx")
	}
	if a.Distribution.UVX != nil {
		types = append(types, "uvx")
	}
	return types
}

// DistributionTypesString returns a comma-separated list of distribution formats.
func (a *Agent) DistributionTypesString() string {
	types := a.DistributionTypes()
	if len(types) == 0 {
		return "none"
	}
	return strings.Join(types, ", ")
}

// SupportsPlatform reports whether the agent can run on the target platform.
func (a *Agent) SupportsPlatform(platform string) bool {
	if a.Distribution.NPX != nil || a.Distribution.UVX != nil {
		return true
	}
	if a.Distribution.Binary != nil {
		_, ok := a.Distribution.Binary[platform]
		return ok
	}
	return false
}

// GetBinaryTarget returns the BinaryTarget for a specific platform if available.
func (a *Agent) GetBinaryTarget(platform string) (*BinaryTarget, error) {
	if a.Distribution.Binary == nil {
		return nil, fmt.Errorf("agent %q does not provide binary distribution", a.ID)
	}
	target, ok := a.Distribution.Binary[platform]
	if !ok {
		return nil, fmt.Errorf("agent %q does not support platform %q", a.ID, platform)
	}
	return &target, nil
}
