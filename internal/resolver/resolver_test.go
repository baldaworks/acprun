package resolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/baldaworks/acprun/internal/registry"
)

func TestResolveNPX(t *testing.T) {
	agent := &registry.Agent{
		ID:      "gemini",
		Version: "0.57.0",
		Distribution: registry.Distribution{
			NPX: &registry.NPXTarget{
				Package: "@google/gemini-cli@0.57.0",
				Args:    []string{"--acp"},
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
		},
	}

	res := NewResolver(nil, nil)
	cmd, err := res.Resolve(context.Background(), agent, ResolveOptions{
		ExtraArgs: []string{"--verbose"},
		ExtraEnv:  map[string]string{"EXTRA": "1"},
	})
	if err != nil {
		t.Fatalf("Resolve NPX failed: %v", err)
	}

	if cmd.Format != "npx" {
		t.Errorf("expected format npx, got %s", cmd.Format)
	}
	if cmd.Executable != "npx" {
		t.Errorf("expected executable npx, got %s", cmd.Executable)
	}

	expectedArgs := []string{"-y", "@google/gemini-cli@0.57.0", "--acp", "--verbose"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, cmd.Args)
	}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("arg[%d] mismatch: got %s, want %s", i, cmd.Args[i], arg)
		}
	}

	if cmd.Env["FOO"] != "BAR" || cmd.Env["EXTRA"] != "1" {
		t.Errorf("unexpected env: %v", cmd.Env)
	}
}

func TestResolveUVX(t *testing.T) {
	agent := &registry.Agent{
		ID:      "fast-agent",
		Version: "0.10.1",
		Distribution: registry.Distribution{
			UVX: &registry.UVXTarget{
				Package: "fast-agent-acp==0.10.1",
				Args:    []string{"-x"},
				Env: map[string]string{
					"FAST_AGENT_MODEL": "codexplan",
				},
			},
		},
	}

	res := NewResolver(nil, nil)
	cmd, err := res.Resolve(context.Background(), agent, ResolveOptions{})
	if err != nil {
		t.Fatalf("Resolve UVX failed: %v", err)
	}

	if cmd.Format != "uvx" {
		t.Errorf("expected format uvx, got %s", cmd.Format)
	}
	if cmd.Executable != "uvx" {
		t.Errorf("expected executable uvx, got %s", cmd.Executable)
	}

	expectedArgs := []string{"fast-agent-acp==0.10.1", "-x"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, cmd.Args)
	}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("arg[%d] mismatch: got %s, want %s", i, cmd.Args[i], arg)
		}
	}
	if cmd.Env["FAST_AGENT_MODEL"] != "codexplan" {
		t.Errorf("unexpected env: %v", cmd.Env)
	}
}

func TestResolveBinary(t *testing.T) {
	// Create dummy binary archive
	zipFile := createTestZip(t, map[string]string{
		"amp-acp": "#!/bin/sh\necho amp\n",
	})
	defer os.Remove(zipFile)

	zipBytes, err := os.ReadFile(zipFile)
	if err != nil {
		t.Fatalf("failed to read zip file: %v", err)
	}
	hasher := sha256.New()
	hasher.Write(zipBytes)
	validSHA256 := hex.EncodeToString(hasher.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "acprun-resolve-bin-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cm := registry.NewCacheManager(tempDir)
	res := NewResolver(cm, nil)

	agent := &registry.Agent{
		ID:      "amp-acp",
		Version: "0.9.0",
		Distribution: registry.Distribution{
			Binary: map[string]registry.BinaryTarget{
				"linux-x86_64": {
					Archive: server.URL + "/amp.zip",
					Cmd:     "./amp-acp",
					Args:    []string{"--server"},
					SHA256:  validSHA256,
				},
			},
		},
	}

	cmd, err := res.Resolve(context.Background(), agent, ResolveOptions{
		Platform:  "linux-x86_64",
		ExtraArgs: []string{"--custom-flag"},
	})
	if err != nil {
		t.Fatalf("Resolve Binary failed: %v", err)
	}

	if cmd.Format != "binary" {
		t.Errorf("expected format binary, got %s", cmd.Format)
	}
	expectedExec := filepath.Join(tempDir, "amp-acp", "0.9.0", "amp-acp")
	if cmd.Executable != expectedExec {
		t.Errorf("expected executable %s, got %s", expectedExec, cmd.Executable)
	}

	expectedArgs := []string{"--server", "--custom-flag"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, cmd.Args)
	}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("arg[%d] mismatch: got %s, want %s", i, cmd.Args[i], arg)
		}
	}

	// Verify cached without re-downloading
	cmdCached, err := res.Resolve(context.Background(), agent, ResolveOptions{
		Platform:   "linux-x86_64",
		NoDownload: true,
	})
	if err != nil {
		t.Fatalf("Resolve with NoDownload failed on cached binary: %v", err)
	}
	if cmdCached.Executable != expectedExec {
		t.Errorf("expected %s, got %s", expectedExec, cmdCached.Executable)
	}

	// Test SHA256 mismatch rejection
	agentBadSHA := &registry.Agent{
		ID:      "amp-bad",
		Version: "0.9.0",
		Distribution: registry.Distribution{
			Binary: map[string]registry.BinaryTarget{
				"linux-x86_64": {
					Archive: server.URL + "/amp.zip",
					Cmd:     "./amp-acp",
					SHA256:  "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
				},
			},
		},
	}

	_, err = res.Resolve(context.Background(), agentBadSHA, ResolveOptions{
		Platform: "linux-x86_64",
	})
	if err == nil {
		t.Errorf("expected SHA256 mismatch error, got nil")
	}
}
