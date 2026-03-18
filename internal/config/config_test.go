package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Default(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != DefaultServerURL {
		t.Errorf("got %q, want %q", cfg.ServerURL, DefaultServerURL)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ServerURL:         "https://custom.example.com",
		DefaultCollection: "my-docs",
		Output:            "json",
	}
	if err := cfg.SaveTo(dir); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ServerURL != "https://custom.example.com" {
		t.Errorf("got %q", loaded.ServerURL)
	}
	if loaded.DefaultCollection != "my-docs" {
		t.Errorf("got %q", loaded.DefaultCollection)
	}
}

func TestSaveConfig_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ServerURL: "https://example.com"}
	if err := cfg.SaveTo(dir); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("got permissions %o, want 0600", info.Mode().Perm())
	}
}
