package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/redpine-ai/connect-cli/internal/mcp"
)

const (
	defaultTTL    = 2 * time.Minute
	cacheFileName = "tools.json"
)

var (
	ErrCacheMiss    = errors.New("cache miss")
	ErrCacheExpired = errors.New("cache expired")
)

type cachedTools struct {
	Timestamp time.Time  `json:"timestamp"`
	Tools     []mcp.Tool `json:"tools"`
}

type ToolCache struct {
	dir string
	ttl time.Duration
}

func NewToolCache(dir string) *ToolCache {
	return &ToolCache{dir: dir, ttl: defaultTTL}
}

func NewToolCacheWithTTL(dir string, ttl time.Duration) *ToolCache {
	return &ToolCache{dir: dir, ttl: ttl}
}

func (tc *ToolCache) Load() ([]mcp.Tool, error) {
	path := filepath.Join(tc.dir, cacheFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	var cached cachedTools
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, ErrCacheMiss
	}

	if time.Since(cached.Timestamp) > tc.ttl {
		return nil, ErrCacheExpired
	}

	return cached.Tools, nil
}

func (tc *ToolCache) Save(tools []mcp.Tool) error {
	if err := os.MkdirAll(tc.dir, 0700); err != nil {
		return err
	}

	cached := cachedTools{
		Timestamp: time.Now(),
		Tools:     tools,
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	// Atomic write
	path := filepath.Join(tc.dir, cacheFileName)
	tmp, err := os.CreateTemp(tc.dir, ".tools-cache-*")
	if err != nil {
		return os.WriteFile(path, data, 0600) // fallback
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()
	return os.Rename(tmpName, path)
}
