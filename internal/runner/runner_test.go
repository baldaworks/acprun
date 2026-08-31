package runner

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/baldaworks/acprun/internal/resolver"
)

func TestRunnerExecuteSuccess(t *testing.T) {
	// Look for a standard executable available in test environment (e.g. echo or go)
	executable, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo binary not available")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resCmd := &resolver.ResolvedCommand{
		AgentID:    "test-agent",
		Version:    "1.0.0",
		Format:     "binary",
		Executable: executable,
		Args:       []string{"hello", "acprun"},
		Env: map[string]string{
			"ACPRUN_TEST": "1",
		},
	}

	exitCode, err := r.Run(context.Background(), resCmd)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "hello acprun" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestRunnerNonZeroExitCode(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh binary not available")
	}

	r := &Runner{}
	resCmd := &resolver.ResolvedCommand{
		AgentID:    "test-agent",
		Version:    "1.0.0",
		Format:     "binary",
		Executable: shPath,
		Args:       []string{"-c", "exit 42"},
	}

	exitCode, err := r.Run(context.Background(), resCmd)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh binary not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := &Runner{}
	resCmd := &resolver.ResolvedCommand{
		AgentID:    "test-agent",
		Version:    "1.0.0",
		Format:     "binary",
		Executable: shPath,
		Args:       []string{"-c", "sleep 5"},
	}

	exitCode, _ := r.Run(ctx, resCmd)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code on cancelled context, got %d", exitCode)
	}
}

func TestRunnerStderrStreamForwarding(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh binary not available")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resCmd := &resolver.ResolvedCommand{
		AgentID:    "test-agent",
		Version:    "1.0.0",
		Format:     "binary",
		Executable: shPath,
		Args:       []string{"-c", "echo 'error message' >&2"},
	}

	exitCode, err := r.Run(context.Background(), resCmd)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}

	stderrOut := strings.TrimSpace(stderr.String())
	if stderrOut != "error message" {
		t.Errorf("expected stderr to contain 'error message', got %q", stderrOut)
	}
}

