package auth

import (
	"bytes"
	"testing"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
)

func TestSetKey_StoresToken(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ios := &output.IOStreams{Out: &stdout, ErrOut: &stderr}
	dir := t.TempDir()
	kr := &mockKeyring{}
	cfg := &config.Config{ServerURL: "https://test.example.com"}

	f := factory.NewTest(ios, cfg, kr)
	cmd := NewSetKeyCmd(f, dir)
	cmd.SetArgs([]string{"sk_live_test123"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if kr.stored != "sk_live_test123" {
		t.Errorf("keyring token = %q", kr.stored)
	}
}

type mockKeyring struct {
	stored string
	err    error
}

func (m *mockKeyring) Get() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.stored, nil
}

func (m *mockKeyring) Set(token string) error {
	if m.err != nil {
		return m.err
	}
	m.stored = token
	return nil
}

func (m *mockKeyring) Delete() error {
	m.stored = ""
	return m.err
}
