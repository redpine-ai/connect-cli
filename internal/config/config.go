package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultServerURL = "https://api.redpine.ai"
	configFileName   = "config.json"
)

// Environment URLs — not exposed in help
var EnvURLs = map[string]string{
	"production": "https://api.redpine.ai",
	"staging":    "https://api-staging.redpine.ai",
}

type Config struct {
	ServerURL         string `json:"server_url"`
	DefaultCollection string `json:"default_collection,omitempty"`
	Output            string `json:"output,omitempty"`
	Environment       string `json:"environment,omitempty"`
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
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
