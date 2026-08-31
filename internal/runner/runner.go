// Package runner handles spawning and lifecycle management of resolved ACP agent processes.
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/baldaworks/acprun/internal/resolver"
)

// Runner manages execution of resolved commands.
type Runner struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	WorkingDir string
}

// NewRunner creates a default Runner attached to OS standard streams.
func NewRunner() *Runner {
	return &Runner{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the resolved command, forwarding standard streams, signals, and returning the exit code.
func (r *Runner) Run(ctx context.Context, resCmd *resolver.ResolvedCommand) (int, error) {
	if resCmd == nil {
		return 1, fmt.Errorf("cannot run nil command")
	}

	execCmd := exec.CommandContext(ctx, resCmd.Executable, resCmd.Args...)

	// Configure working directory
	if r.WorkingDir != "" {
		execCmd.Dir = r.WorkingDir
	} else if resCmd.WorkingDir != "" {
		execCmd.Dir = resCmd.WorkingDir
	}

	// Merge environment variables
	envMap := make(map[string]string)
	for _, envStr := range os.Environ() {
		for i := 0; i < len(envStr); i++ {
			if envStr[i] == '=' {
				envMap[envStr[:i]] = envStr[i+1:]
				break
			}
		}
	}
	for k, v := range resCmd.Env {
		envMap[k] = v
	}

	var envList []string
	for k, v := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	execCmd.Env = envList

	// Attach streams
	if r.Stdin != nil {
		execCmd.Stdin = r.Stdin
	} else {
		execCmd.Stdin = os.Stdin
	}
	if r.Stdout != nil {
		execCmd.Stdout = r.Stdout
	} else {
		execCmd.Stdout = os.Stdout
	}
	if r.Stderr != nil {
		execCmd.Stderr = r.Stderr
	} else {
		execCmd.Stderr = os.Stderr
	}

	// Start process
	if err := execCmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start process %s: %w", resCmd.Executable, err)
	}

	// Forward OS signals to child process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

	go func() {
		for sig := range sigChan {
			if execCmd.Process != nil {
				_ = execCmd.Process.Signal(sig)
			}
		}
	}()

	// Wait for process termination
	err := execCmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		if ctx.Err() != nil {
			return 130, ctx.Err()
		}
		return 1, err
	}

	return 0, nil
}
