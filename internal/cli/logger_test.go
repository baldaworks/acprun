package cli

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestInitLoggerLevels(t *testing.T) {
	buf := new(bytes.Buffer)

	// Non-verbose: Debug messages should be suppressed
	InitLoggerWithWriter(buf, false)
	slog.Debug("debug message non-verbose", "key", "val")
	slog.Info("info message non-verbose", "key", "val")

	output := buf.String()
	if strings.Contains(output, "debug message non-verbose") {
		t.Errorf("expected debug message to be suppressed in non-verbose mode, got: %s", output)
	}
	if !strings.Contains(output, "info message non-verbose") {
		t.Errorf("expected info message to be present, got: %s", output)
	}

	buf.Reset()

	// Verbose: Debug messages should be logged
	InitLoggerWithWriter(buf, true)
	slog.Debug("debug message verbose", "key", "val")
	slog.Info("info message verbose", "key", "val")

	outputVerbose := buf.String()
	if !strings.Contains(outputVerbose, "debug message verbose") {
		t.Errorf("expected debug message in verbose mode, got: %s", outputVerbose)
	}
	if !strings.Contains(outputVerbose, "info message verbose") {
		t.Errorf("expected info message in verbose mode, got: %s", outputVerbose)
	}
}
