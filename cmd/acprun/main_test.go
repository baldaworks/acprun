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

	"github.com/baldaworks/acprun/internal/cli"
	"github.com/baldaworks/acprun/internal/registry"
	"github.com/baldaworks/acprun/internal/resolver"
	"github.com/baldaworks/acprun/internal/runner"
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

func TestEndToEndStreamIsolationAndStructuredLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "acprun-logging-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockScript := `#!/bin/sh
echo '{"jsonrpc":"2.0","method":"initialized"}'
echo 'AGENT_STDERR_LOG: agent diagnostic' >&2
`
	zipData := createTestZipBytes(t, map[string]string{
		"mock-acp-agent": mockScript,
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
						ID:          "mock-acp-agent",
						Name:        "Mock ACP Agent",
						Version:     "2.0.0",
						Description: "Logging Test Mock Agent",
						Distribution: registry.Distribution{
							Binary: map[string]registry.BinaryTarget{
								"linux-x86_64": {
									Archive: server.URL + "/agent.zip",
									Cmd:     "./mock-acp-agent",
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
		if strings.HasSuffix(r.URL.Path, "agent.zip") {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	var stdoutBuf, stderrBuf bytes.Buffer

	// Configure logger to write to stderrBuf in verbose mode
	cli.InitLoggerWithWriter(&stderrBuf, true)

	cacheDir := filepath.Join(tempDir, "cache")
	client := registry.NewClient(registry.ClientOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    cacheDir,
	})
	res := resolver.NewResolver(client.CacheManager(), nil)

	ctx := context.Background()
	agent, err := client.GetAgent(ctx, "mock-acp-agent")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	resolved, err := res.Resolve(ctx, agent, resolver.ResolveOptions{
		Platform: "linux-x86_64",
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	r := &runner.Runner{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	exitCode, err := r.Run(ctx, resolved)
	if err != nil {
		t.Fatalf("Runner.Run failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// 1. Stdout must contain ONLY the JSON-RPC agent output, zero acprun logs
	stdoutStr := strings.TrimSpace(stdoutBuf.String())
	expectedStdout := `{"jsonrpc":"2.0","method":"initialized"}`
	if stdoutStr != expectedStdout {
		t.Errorf("expected stdout to strictly contain %q, got %q", expectedStdout, stdoutStr)
	}
	if strings.Contains(stdoutStr, "level=") || strings.Contains(stdoutStr, "msg=") {
		t.Errorf("stdout contains slog log output! %q", stdoutStr)
	}

	// 2. Stderr must contain the agent's stderr output AND acprun's structured logs
	stderrStr := stderrBuf.String()
	if !strings.Contains(stderrStr, "AGENT_STDERR_LOG: agent diagnostic") {
		t.Errorf("expected stderr to contain agent stderr diagnostic, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "level=INFO") || !strings.Contains(stderrStr, "downloading agent binary archive") {
		t.Errorf("expected stderr to contain structured slog logs, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "level=DEBUG") || !strings.Contains(stderrStr, "spawning agent process") {
		t.Errorf("expected stderr to contain verbose debug logs, got: %s", stderrStr)
	}
}

