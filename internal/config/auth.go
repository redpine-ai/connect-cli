package config

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const credsFileName = "credentials.json"

var ErrKeyringUnavailable = errors.New("keyring unavailable")

type Credentials struct {
	Token        string `json:"token"`
	Type         string `json:"type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	ServerURL    string `json:"server_url,omitempty"`
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

// RefreshOAuthToken uses a stored refresh token to get a new access token.
// Updates credentials in place (both keyring and file). Returns the new access token.
func RefreshOAuthToken(kr Keyring) (string, error) {
	dir := ConfigDir()
	creds, err := LoadCredentialsFrom(dir)
	if err != nil || creds.RefreshToken == "" || creds.ClientID == "" {
		return "", errors.New("no refresh token available — run 'connect auth login'")
	}

	serverURL := creds.ServerURL
	if serverURL == "" {
		serverURL = DefaultServerURL
	}

	resp, err := http.PostForm(serverURL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {creds.RefreshToken},
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("refresh token expired — run 'connect auth login'")
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Update stored credentials with new tokens
	creds.Token = result.AccessToken
	creds.RefreshToken = result.RefreshToken

	// Save to keyring
	if kr != nil {
		kr.Set(creds.Token)
	}
	// Always update file (has refresh token)
	creds.SaveTo(dir)

	return result.AccessToken, nil
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
