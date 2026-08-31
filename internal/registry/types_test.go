package registry

import (
	"encoding/json"
	"testing"
)

const sampleJSON = `{
  "version": "1.0.0",
  "agents": [
    {
      "id": "amp-acp",
      "name": "Amp",
      "version": "0.9.0",
      "description": "ACP wrapper for Amp",
      "authors": ["tao"],
      "license": "Apache-2.0",
      "distribution": {
        "binary": {
          "linux-x86_64": {
            "archive": "https://example.com/amp-linux-x86_64.tar.gz",
            "cmd": "./amp-acp",
            "sha256": "afaa50a152eb86a8ff21e354ded63fe2d21b730859692e3a60b2c4c9ef23df31"
          }
        }
      }
    },
    {
      "id": "gemini",
      "name": "Gemini CLI",
      "version": "0.57.0",
      "distribution": {
        "npx": {
          "package": "@google/gemini-cli@0.57.0",
          "args": ["--acp"]
        }
      }
    },
    {
      "id": "fast-agent",
      "name": "fast-agent",
      "version": "0.10.1",
      "distribution": {
        "uvx": {
          "package": "fast-agent-acp==0.10.1",
          "args": ["-x"],
          "env": {
            "FAST_AGENT_MODEL": "codexplan"
          }
        }
      }
    }
  ]
}`

func TestRegistryUnmarshal(t *testing.T) {
	var reg Registry
	if err := json.Unmarshal([]byte(sampleJSON), &reg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if reg.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", reg.Version)
	}
	if len(reg.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(reg.Agents))
	}

	amp := reg.Agents[0]
	if amp.ID != "amp-acp" {
		t.Errorf("expected amp-acp, got %s", amp.ID)
	}
	if types := amp.DistributionTypes(); len(types) != 1 || types[0] != "binary" {
		t.Errorf("expected [binary], got %v", types)
	}
	if !amp.SupportsPlatform("linux-x86_64") {
		t.Errorf("expected amp to support linux-x86_64")
	}
	if amp.SupportsPlatform("darwin-arm64") {
		t.Errorf("did not expect amp to support darwin-arm64 in sample")
	}

	target, err := amp.GetBinaryTarget("linux-x86_64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Cmd != "./amp-acp" {
		t.Errorf("expected ./amp-acp, got %s", target.Cmd)
	}
	if target.SHA256 != "afaa50a152eb86a8ff21e354ded63fe2d21b730859692e3a60b2c4c9ef23df31" {
		t.Errorf("unexpected sha256: %s", target.SHA256)
	}

	gemini := reg.Agents[1]
	if gemini.DistributionTypesString() != "npx" {
		t.Errorf("expected npx, got %s", gemini.DistributionTypesString())
	}
	if !gemini.SupportsPlatform("any-platform") {
		t.Errorf("npx agent should support any platform")
	}

	fastAgent := reg.Agents[2]
	if fastAgent.DistributionTypesString() != "uvx" {
		t.Errorf("expected uvx, got %s", fastAgent.DistributionTypesString())
	}
	if fastAgent.Distribution.UVX.Env["FAST_AGENT_MODEL"] != "codexplan" {
		t.Errorf("unexpected env var: %v", fastAgent.Distribution.UVX.Env)
	}
}
