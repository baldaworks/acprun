package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "acprun-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cm := NewCacheManager(tempDir)
	if cm.RootDir() != tempDir {
		t.Errorf("expected %s, got %s", tempDir, cm.RootDir())
	}

	manifestPath := cm.ManifestPath()
	if manifestPath != filepath.Join(tempDir, "manifest.json") {
		t.Errorf("unexpected manifest path: %s", manifestPath)
	}

	agentDir := cm.AgentDir("amp-acp", "0.9.0")
	expectedAgentDir := filepath.Join(tempDir, "amp-acp", "0.9.0")
	if agentDir != expectedAgentDir {
		t.Errorf("expected %s, got %s", expectedAgentDir, agentDir)
	}

	if err := cm.EnsureDir(agentDir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create test manifest file
	if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	// Test Clean ManifestsOnly
	if err := cm.Clean(CleanOptions{ManifestsOnly: true}); err != nil {
		t.Fatalf("Clean ManifestsOnly failed: %v", err)
	}
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Errorf("manifest should have been deleted")
	}

	// Test Clean All
	if err := cm.Clean(CleanOptions{All: true}); err != nil {
		t.Fatalf("Clean All failed: %v", err)
	}
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("tempDir should have been deleted")
	}
}
