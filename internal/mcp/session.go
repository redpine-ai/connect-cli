package mcp

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SessionCache struct {
	path string
}

func NewSessionCache(dir, serverURL string) *SessionCache {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(serverURL)))
	return &SessionCache{
		path: filepath.Join(dir, "redpine-session-"+hash[:16]),
	}
}

func DefaultSessionCache(serverURL string) *SessionCache {
	return NewSessionCache(os.TempDir(), serverURL)
}

func (sc *SessionCache) Load() string {
	data, err := os.ReadFile(sc.path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (sc *SessionCache) Save(sessionID string) {
	_ = os.WriteFile(sc.path, []byte(sessionID), 0600) // best-effort session cache
}

func (sc *SessionCache) Delete() {
	_ = os.Remove(sc.path) // best-effort cleanup
}
