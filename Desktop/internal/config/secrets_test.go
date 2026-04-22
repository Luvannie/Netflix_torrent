package config

import (
	"errors"
	"path/filepath"
	"testing"
)

type fakeProtector struct {
	protected    []byte
	unprotected  []byte
	protectErr   error
	unprotectErr error
}

func (p fakeProtector) Protect(data []byte) ([]byte, error) {
	if p.protectErr != nil {
		return nil, p.protectErr
	}
	return append([]byte(nil), p.protected...), nil
}

func (p fakeProtector) Unprotect(data []byte) ([]byte, error) {
	if p.unprotectErr != nil {
		return nil, p.unprotectErr
	}
	return append([]byte(nil), p.unprotected...), nil
}

func TestFileSecretStoreSavesAndLoadsProtectedSecret(t *testing.T) {
	dir := t.TempDir()
	store := NewFileSecretStore(filepath.Join(dir, "secrets"), fakeProtector{
		protected:   []byte("cipher"),
		unprotected: []byte("secret-value"),
	})

	if err := store.Save("local-token", "secret-value"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load("local-token")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("Load() = %q", got)
	}
}

func TestFileSecretStorePropagatesProtectError(t *testing.T) {
	dir := t.TempDir()
	store := NewFileSecretStore(filepath.Join(dir, "secrets"), fakeProtector{
		protectErr: errors.New("protect failed"),
	})

	err := store.Save("local-token", "secret-value")
	if err == nil {
		t.Fatalf("expected error")
	}
}
