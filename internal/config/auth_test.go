package config

import (
	"os"
	"testing"
)

func TestResolveToken_FlagFirst(t *testing.T) {
	token, source := ResolveToken("sk_flag", nil)
	if token != "sk_flag" || source != "flag" {
		t.Errorf("got token=%q source=%q", token, source)
	}
}

func TestResolveToken_EnvSecond(t *testing.T) {
	os.Setenv("CONNECT_API_KEY", "sk_env")
	defer os.Unsetenv("CONNECT_API_KEY")

	token, source := ResolveToken("", nil)
	if token != "sk_env" || source != "env" {
		t.Errorf("got token=%q source=%q", token, source)
	}
}

func TestResolveToken_CredentialsFallback(t *testing.T) {
	dir := t.TempDir()
	creds := &Credentials{Token: "sk_file", Type: "api_key"}
	if err := creds.SaveTo(dir); err != nil {
		t.Fatal(err)
	}

	kr := &mockKeyring{err: ErrKeyringUnavailable}
	token, source := resolveTokenFrom("", kr, dir)
	if token != "sk_file" || source != "file" {
		t.Errorf("got token=%q source=%q", token, source)
	}
}

type mockKeyring struct {
	token string
	err   error
}

func (m *mockKeyring) Get() (string, error)  { return m.token, m.err }
func (m *mockKeyring) Set(token string) error { return m.err }
func (m *mockKeyring) Delete() error          { return m.err }
