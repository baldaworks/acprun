package resolver

import (
	"fmt"

	"github.com/baldaworks/acprun/internal/registry"
)

func (r *Resolver) resolveNPX(agent *registry.Agent) (*ResolvedCommand, error) {
	if agent.Distribution.NPX == nil {
		return nil, fmt.Errorf("agent %q does not provide NPX distribution", agent.ID)
	}

	target := agent.Distribution.NPX
	if target.Package == "" {
		return nil, fmt.Errorf("agent %q NPX target is missing package name", agent.ID)
	}

	args := []string{"-y", target.Package}
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
		Format:     "npx",
		Executable: "npx",
		Args:       args,
		Env:        env,
	}, nil
}
