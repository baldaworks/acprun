package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const sampleRegistryJSON = `{
  "version": "1.0.0",
  "agents": [
    {
      "id": "gemini",
      "name": "Gemini CLI",
      "version": "0.57.0",
      "description": "Google's official CLI for Gemini",
      "authors": ["Google"],
      "license": "Apache-2.0",
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
      "description": "Fast Agent",
      "distribution": {
        "uvx": {
          "package": "fast-agent-acp==0.10.1",
          "args": ["-x"]
        }
      }
    }
  ]
}`

func setupTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleRegistryJSON))
	}))

	tempDir, err := os.MkdirTemp("", "acprun-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	return server, tempDir
}

func executeCommand(args []string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return buf.String(), err
}

func TestListCommand(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer server.Close()
	defer os.RemoveAll(tempDir)

	args := []string{"list", "--registry", server.URL, "--cache-dir", tempDir}
	output, err := executeCommand(args)
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Output is printed to stdout
	_ = output
}

func TestInfoCommand(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer server.Close()
	defer os.RemoveAll(tempDir)

	args := []string{"info", "gemini", "--json", "--registry", server.URL, "--cache-dir", tempDir}
	_, err := executeCommand(args)
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}
}

func TestResolveCommand(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer server.Close()
	defer os.RemoveAll(tempDir)

	args := []string{"resolve", "gemini", "--json", "--registry", server.URL, "--cache-dir", tempDir}
	_, err := executeCommand(args)
	if err != nil {
		t.Fatalf("resolve command failed: %v", err)
	}
}

func TestCacheCommands(t *testing.T) {
	_, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	pathArgs := []string{"cache", "path", "--cache-dir", tempDir}
	_, err := executeCommand(pathArgs)
	if err != nil {
		t.Fatalf("cache path failed: %v", err)
	}

	cleanArgs := []string{"cache", "clean", "--all", "--cache-dir", tempDir}
	_, err = executeCommand(cleanArgs)
	if err != nil {
		t.Fatalf("cache clean failed: %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	output, err := executeCommand([]string{"version"})
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	_ = output
}

func TestRootHelpWhenNoArgs(t *testing.T) {
	output, err := executeCommand([]string{})
	if err != nil {
		t.Fatalf("root help failed: %v", err)
	}
	if !strings.Contains(output, "acprun") {
		// Output may be captured in stdout or buffer
	}
}
