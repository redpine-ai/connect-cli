package config

import (
	"testing"
	"time"
)

func TestNoopKeyring_ReturnsUnavailable(t *testing.T) {
	kr := &noopKeyring{}

	_, err := kr.Get()
	if err != ErrKeyringUnavailable {
		t.Errorf("Get: got %v, want ErrKeyringUnavailable", err)
	}

	err = kr.Set("token")
	if err != ErrKeyringUnavailable {
		t.Errorf("Set: got %v, want ErrKeyringUnavailable", err)
	}

	err = kr.Delete()
	if err != ErrKeyringUnavailable {
		t.Errorf("Delete: got %v, want ErrKeyringUnavailable", err)
	}
}

func TestWithTimeout_Success(t *testing.T) {
	val, err := withTimeout(func() (string, error) {
		return "hello", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %q, want %q", val, "hello")
	}
}

func TestWithTimeout_Timeout(t *testing.T) {
	val, err := withTimeout(func() (string, error) {
		time.Sleep(5 * time.Second)
		return "late", nil
	})
	if err != ErrKeyringUnavailable {
		t.Errorf("got err=%v, want ErrKeyringUnavailable", err)
	}
	if val != "" {
		t.Errorf("got val=%q, want empty", val)
	}
}

func TestNewSystemKeyring_NotNil(t *testing.T) {
	kr := NewSystemKeyring()
	if kr == nil {
		t.Error("expected non-nil keyring")
	}
}
