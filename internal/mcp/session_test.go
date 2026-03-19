package mcp

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
)

func TestSessionCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	sc := NewSessionCache(dir, "https://api.example.com")

	sc.Save("session-123")

	loaded := sc.Load()
	if loaded != "session-123" {
		t.Errorf("got %q", loaded)
	}
}

func TestSessionCache_Missing(t *testing.T) {
	dir := t.TempDir()
	sc := NewSessionCache(dir, "https://api.example.com")

	loaded := sc.Load()
	if loaded != "" {
		t.Errorf("expected empty, got %q", loaded)
	}
}

func TestSessionCache_Delete(t *testing.T) {
	dir := t.TempDir()
	sc := NewSessionCache(dir, "https://api.example.com")

	sc.Save("session-123")
	sc.Delete()

	if sc.Load() != "" {
		t.Error("should be empty after delete")
	}
}

func TestSessionCache_FileNameIsHash(t *testing.T) {
	dir := t.TempDir()
	sc := NewSessionCache(dir, "https://api.example.com")

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("https://api.example.com")))
	expected := filepath.Join(dir, "redpine-session-"+hash[:16])

	if sc.path != expected {
		t.Errorf("path = %q, want %q", sc.path, expected)
	}
}
