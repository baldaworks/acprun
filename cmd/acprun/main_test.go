package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baldaworks/acprun/internal/registry"
	"github.com/baldaworks/acprun/internal/resolver"
)

func createTestZipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for name, content := range files {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		header.SetMode(0755)
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create zip header: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip content: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestEndToEndBinaryResolutionAndRun(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "acprun-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	zipData := createTestZipBytes(t, map[string]string{
		"mock-agent": "#!/bin/sh\necho \"MOCK_AGENT_OUTPUT: $1 $2\"\n",
	})

	hasher := sha256.New()
	hasher.Write(zipData)
	sha := hex.EncodeToString(hasher.Sum(nil))

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "registry.json") {
			reg := registry.Registry{
				Version: "1.0.0",
				Agents: []registry.Agent{
					{
						ID:          "mock-agent",
						Name:        "Mock Agent",
						Version:     "1.0.0",
						Description: "E2E Test Mock Agent",
						Distribution: registry.Distribution{
							Binary: map[string]registry.BinaryTarget{
								"linux-x86_64": {
									Archive: server.URL + "/mock-agent.zip",
									Cmd:     "./mock-agent",
									Args:    []string{"default-arg"},
									SHA256:  sha,
								},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(reg)
			return
		}

		if strings.HasSuffix(r.URL.Path, "mock-agent.zip") {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cacheDir := filepath.Join(tempDir, "cache")
	client := registry.NewClient(registry.ClientOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    cacheDir,
	})
	res := resolver.NewResolver(client.CacheManager(), nil)

	ctx := context.Background()
	agent, err := client.GetAgent(ctx, "mock-agent")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	resolved, err := res.Resolve(ctx, agent, resolver.ResolveOptions{
		Platform:  "linux-x86_64",
		ExtraArgs: []string{"extra-val"},
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Format != "binary" {
		t.Errorf("expected format binary, got %s", resolved.Format)
	}

	// Verify the extracted binary exists and can be executed
	cmd := exec.Command(resolved.Executable, resolved.Args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execution of resolved binary failed: %v, output: %s", err, string(out))
	}

	if !strings.Contains(string(out), "MOCK_AGENT_OUTPUT: default-arg extra-val") {
		t.Errorf("unexpected output from mock agent: %s", string(out))
	}
}
