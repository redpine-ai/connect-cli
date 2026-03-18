package cache

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/redpine-ai/connect-cli/internal/mcp"
)

func TestToolCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cache := NewToolCache(dir)

	tools := []mcp.Tool{
		{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{}`)},
		{Name: "analytics--query", Description: "Run query", InputSchema: json.RawMessage(`{}`)},
	}

	if err := cache.Save(tools); err != nil {
		t.Fatal(err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Errorf("got %d tools", len(loaded))
	}
}

func TestToolCache_TTLExpiry(t *testing.T) {
	dir := t.TempDir()
	cache := NewToolCacheWithTTL(dir, 1*time.Millisecond)

	tools := []mcp.Tool{{Name: "test"}}
	cache.Save(tools)

	time.Sleep(5 * time.Millisecond)

	_, err := cache.Load()
	if err != ErrCacheExpired {
		t.Errorf("expected ErrCacheExpired, got %v", err)
	}
}

func TestToolCache_Missing(t *testing.T) {
	dir := t.TempDir()
	cache := NewToolCache(dir)

	_, err := cache.Load()
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}
