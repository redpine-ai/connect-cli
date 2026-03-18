package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const credsFileName = "credentials.json"

var ErrKeyringUnavailable = errors.New("keyring unavailable")

type Credentials struct {
	Token string `json:"token"`
	Type  string `json:"type"`
}

type Keyring interface {
	Get() (string, error)
	Set(token string) error
	Delete() error
}

func (c *Credentials) SaveTo(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, credsFileName), data, 0600)
}

func LoadCredentialsFrom(dir string) (*Credentials, error) {
	path := filepath.Join(dir, credsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func ResolveToken(flagValue string, kr Keyring) (token, source string) {
	return resolveTokenFrom(flagValue, kr, ConfigDir())
}

func resolveTokenFrom(flagValue string, kr Keyring, configDir string) (token, source string) {
	// 1. Flag
	if flagValue != "" {
		return flagValue, "flag"
	}
	// 2. Env var
	if env := os.Getenv("CONNECT_API_KEY"); env != "" {
		return env, "env"
	}
	// 3. Keyring
	if kr != nil {
		if t, err := kr.Get(); err == nil && t != "" {
			return t, "keyring"
		}
	}
	// 4. Credentials file
	creds, err := LoadCredentialsFrom(configDir)
	if err == nil && creds.Token != "" {
		return creds.Token, "file"
	}
	return "", ""
}
