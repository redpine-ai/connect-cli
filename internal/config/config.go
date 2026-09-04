package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultServerURL = "https://api.redpine.ai"
	configFileName   = "config.json"

	// Environment variables. The REDPINE_* names match the SDKs; the
	// CONNECT_* names are the original CLI names and stay as fallbacks.
	// #nosec G101 -- these are variable NAMES, not credential values.
	EnvAPIKey         = "REDPINE_API_KEY" // #nosec G101
	EnvAPIKeyFallback = "CONNECT_API_KEY" // #nosec G101
	EnvBaseURL         = "REDPINE_BASE_URL"
	EnvBaseURLFallback = "CONNECT_SERVER_URL"
)

// Environment URLs — not exposed in help
var EnvURLs = map[string]string{
	"production": "https://api.redpine.ai",
	"staging":    "https://api-staging.redpine.ai",
}

// ServerURLFromEnv returns the server URL override from the environment, or
// "" when neither REDPINE_BASE_URL nor CONNECT_SERVER_URL is set.
func ServerURLFromEnv() string {
	for _, name := range []string{EnvBaseURL, EnvBaseURLFallback} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

type Config struct {
	ServerURL   string `json:"server_url"`
	Environment string `json:"environment,omitempty"`
}

// ServerURLForEnv returns the server URL for the current environment.
func (c *Config) ServerURLForEnv() string {
	if c.Environment != "" {
		if url, ok := EnvURLs[c.Environment]; ok {
			return url
		}
	}
	return c.ServerURL
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "redpine")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "redpine")
}

func Load() (*Config, error) {
	return LoadFrom(ConfigDir())
}

func LoadFrom(dir string) (*Config, error) {
	path := filepath.Join(dir, configFileName)
	// #nosec G304 -- path is the CLI's own config file under its config dir, not user input.
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{ServerURL: DefaultServerURL}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	return c.SaveTo(ConfigDir())
}

func (c *Config) SaveTo(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, configFileName), data, 0600)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".connect-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
