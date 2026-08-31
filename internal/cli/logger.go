package cli

import (
	"io"
	"log/slog"
	"os"
)

// InitLogger configures the global slog default logger to write structured records
// exclusively to the specified writer (or os.Stderr by default) with level based on verbose.
func InitLogger(verbose bool) {
	InitLoggerWithWriter(os.Stderr, verbose)
}

// InitLoggerWithWriter configures slog with a specific output writer and verbose flag.
func InitLoggerWithWriter(w io.Writer, verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(w, opts)
	slog.SetDefault(slog.New(handler))
}
