package resolver

import (
	"fmt"

	"github.com/baldaworks/acprun/internal/registry"
)

func (r *Resolver) resolveUVX(agent *registry.Agent) (*ResolvedCommand, error) {
	if agent.Distribution.UVX == nil {
		return nil, fmt.Errorf("agent %q does not provide UVX distribution", agent.ID)
	}

	target := agent.Distribution.UVX
	if target.Package == "" {
		return nil, fmt.Errorf("agent %q UVX target is missing package name", agent.ID)
	}

	args := []string{target.Package}
	if len(target.Args) > 0 {
		args = append(args, target.Args...)
	}

	env := make(map[string]string)
	for k, v := range target.Env {
		env[k] = v
	}

	return &ResolvedCommand{
		AgentID:    agent.ID,
		Version:    agent.Version,
		Format:     "uvx",
		Executable: "uvx",
		Args:       args,
		Env:        env,
	}, nil
}
