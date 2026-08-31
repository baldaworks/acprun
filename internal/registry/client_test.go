package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientFetchAndCache(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleJSON))
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "acprun-client-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := NewClient(ClientOptions{
		RegistryURL: server.URL,
		CacheDir:    tempDir,
		CacheTTL:    10 * time.Minute,
	})

	ctx := context.Background()

	// First fetch should hit network
	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("FetchRegistry failed: %v", err)
	}
	if len(reg.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(reg.Agents))
	}
	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("expected 1 network request, got %d", count)
	}

	// Second fetch should use cache without hitting network
	client2 := NewClient(ClientOptions{
		RegistryURL: server.URL,
		CacheDir:    tempDir,
		CacheTTL:    10 * time.Minute,
	})
	reg2, err := client2.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("FetchRegistry from cache failed: %v", err)
	}
	if len(reg2.Agents) != 3 {
		t.Fatalf("expected 3 agents from cache, got %d", len(reg2.Agents))
	}
	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("expected requestCount to remain 1, got %d", count)
	}
}

func TestClientOfflineFallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "acprun-client-offline-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Pre-populate cache
	client := clientWithCache(tempDir)
	if err := os.WriteFile(client.CacheManager().ManifestPath(), []byte(sampleJSON), 0644); err != nil {
		t.Fatalf("failed to write test cache: %v", err)
	}

	// Client pointing to an invalid/unreachable URL with offline = true
	offlineClient := NewClient(ClientOptions{
		RegistryURL: "http://unreachable.invalid/registry.json",
		CacheDir:    tempDir,
		Offline:     true,
	})

	reg, err := offlineClient.FetchRegistry(context.Background())
	if err != nil {
		t.Fatalf("expected offline fetch to succeed with cached manifest, got error: %v", err)
	}
	if len(reg.Agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(reg.Agents))
	}

	// Client pointing to unreachable URL without offline flag should fallback to stale cache
	onlineClientWithBrokenNetwork := NewClient(ClientOptions{
		RegistryURL: "http://unreachable.invalid/registry.json",
		CacheDir:    tempDir,
		CacheTTL:    1 * time.Nanosecond, // Force cache expiration
	})

	regFallback, err := onlineClientWithBrokenNetwork.FetchRegistry(context.Background())
	if err != nil {
		t.Fatalf("expected fallback to stale cache to succeed, got error: %v", err)
	}
	if len(regFallback.Agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(regFallback.Agents))
	}
}

func TestClientGetAgentAndList(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "acprun-getagent-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleJSON))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		RegistryURL: server.URL,
		CacheDir:    tempDir,
	})

	ctx := context.Background()

	agent, err := client.GetAgent(ctx, "amp-acp")
	if err != nil {
		t.Fatalf("GetAgent(amp-acp) failed: %v", err)
	}
	if agent.Name != "Amp" {
		t.Errorf("expected Amp, got %s", agent.Name)
	}

	_, err = client.GetAgent(ctx, "nonexistent-agent")
	if err == nil {
		t.Errorf("expected error for nonexistent agent, got nil")
	}

	agents, err := client.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}
}

func clientWithCache(dir string) *Client {
	return NewClient(ClientOptions{CacheDir: dir})
}
