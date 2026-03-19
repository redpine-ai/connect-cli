package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redpine-ai/connect-cli/internal/version"
)

const (
	releasesURL = "https://api.github.com/repos/redpine-ai/connect-cli/releases/latest"
	cacheTTL    = 1 * time.Hour
	cacheFile   = "latest-version.json"
)

type versionCache struct {
	Version   string    `json:"version"`
	CheckedAt time.Time `json:"checked_at"`
}

// CheckResult describes the outcome of a version check.
type CheckResult struct {
	Current    string
	Latest     string
	IsOutdated bool
}

// Check compares the running version against the latest GitHub release.
// Results are cached to avoid hitting the API on every invocation.
func Check(cacheDir string) *CheckResult {
	current := version.Version
	if current == "dev" {
		return nil // dev builds skip version checks
	}

	latest := loadCachedVersion(cacheDir)
	if latest == "" {
		latest = fetchLatestVersion()
		if latest != "" {
			saveCachedVersion(cacheDir, latest)
		}
	}

	if latest == "" {
		return nil // can't reach GitHub, don't block
	}

	return &CheckResult{
		Current:    current,
		Latest:     latest,
		IsOutdated: latest != current,
	}
}

func loadCachedVersion(cacheDir string) string {
	path := filepath.Join(cacheDir, cacheFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cache versionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return ""
	}
	if time.Since(cache.CheckedAt) > cacheTTL {
		// Expired — fetch fresh but return stale for now to avoid blocking
		go func() {
			if v := fetchLatestVersion(); v != "" {
				saveCachedVersion(cacheDir, v)
			}
		}()
		return cache.Version
	}
	return cache.Version
}

func saveCachedVersion(cacheDir, ver string) {
	os.MkdirAll(cacheDir, 0700)
	data, _ := json.Marshal(versionCache{Version: ver, CheckedAt: time.Now()})
	os.WriteFile(filepath.Join(cacheDir, cacheFile), data, 0600)
}

func fetchLatestVersion() string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(releasesURL)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}
	return strings.TrimPrefix(release.TagName, "v")
}

// FormatWarning returns a stderr message for outdated versions.
func (r *CheckResult) FormatWarning() string {
	return fmt.Sprintf("Update required: v%s → v%s. Run: connect update", r.Current, r.Latest)
}
