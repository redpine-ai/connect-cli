package config

import (
	"os"
	"runtime"
	"strings"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

const (
	keyringService = "redpine-cli"
	keyringUser    = "default"
	keyringTimeout = 3 * time.Second
)

type SystemKeyring struct{}

func NewSystemKeyring() Keyring {
	if isWSL() {
		return &noopKeyring{}
	}
	return &SystemKeyring{}
}

func (k *SystemKeyring) Get() (string, error) {
	return withTimeout(func() (string, error) {
		return gokeyring.Get(keyringService, keyringUser)
	})
}

func (k *SystemKeyring) Set(token string) error {
	_, err := withTimeout(func() (string, error) {
		return "", gokeyring.Set(keyringService, keyringUser, token)
	})
	return err
}

func (k *SystemKeyring) Delete() error {
	_, err := withTimeout(func() (string, error) {
		return "", gokeyring.Delete(keyringService, keyringUser)
	})
	return err
}

type noopKeyring struct{}

func (k *noopKeyring) Get() (string, error) { return "", ErrKeyringUnavailable }
func (k *noopKeyring) Set(string) error      { return ErrKeyringUnavailable }
func (k *noopKeyring) Delete() error         { return ErrKeyringUnavailable }

func withTimeout(fn func() (string, error)) (string, error) {
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		v, e := fn()
		ch <- result{v, e}
	}()
	select {
	case r := <-ch:
		return r.val, r.err
	case <-time.After(keyringTimeout):
		return "", ErrKeyringUnavailable
	}
}

func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
