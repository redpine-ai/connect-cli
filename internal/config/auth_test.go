package config

import "testing"

func TestResolveToken_FlagFirst(t *testing.T) {
	token, source := ResolveToken("sk_flag", nil)
	if token != "sk_flag" || source != "flag" {
		t.Errorf("got token=%q source=%q", token, source)
	}
}

func TestResolveToken_EnvSecond(t *testing.T) {
	t.Setenv("REDPINE_API_KEY", "")
	t.Setenv("CONNECT_API_KEY", "sk_env")

	token, source := ResolveToken("", nil)
	if token != "sk_env" || source != "env" {
		t.Errorf("got token=%q source=%q", token, source)
	}
}

// The SDKs read REDPINE_API_KEY first and fall back to CONNECT_API_KEY; the
// CLI must resolve identically or the two disagree on the same shell.
func TestResolveToken_RedpineEnvWinsOverLegacy(t *testing.T) {
	t.Setenv("REDPINE_API_KEY", "sk_redpine")
	t.Setenv("CONNECT_API_KEY", "sk_legacy")

	token, source := ResolveToken("", nil)
	if token != "sk_redpine" || source != "env" {
		t.Errorf("got token=%q source=%q, want REDPINE_API_KEY to win", token, source)
	}
}

func TestServerURLFromEnv(t *testing.T) {
	t.Setenv("REDPINE_BASE_URL", "")
	t.Setenv("CONNECT_SERVER_URL", "")
	if got := ServerURLFromEnv(); got != "" {
		t.Errorf("unset: got %q", got)
	}
	t.Setenv("CONNECT_SERVER_URL", "https://legacy.example")
	if got := ServerURLFromEnv(); got != "https://legacy.example" {
		t.Errorf("legacy only: got %q", got)
	}
	t.Setenv("REDPINE_BASE_URL", "https://new.example")
	if got := ServerURLFromEnv(); got != "https://new.example" {
		t.Errorf("both set: got %q, want REDPINE_BASE_URL to win", got)
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

func (m *mockKeyring) Get() (string, error)   { return m.token, m.err }
func (m *mockKeyring) Set(token string) error { return m.err }
func (m *mockKeyring) Delete() error          { return m.err }
