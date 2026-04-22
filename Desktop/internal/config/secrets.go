package config

import (
	"errors"
	"os"
	"path/filepath"
)

type SecretProtector interface {
	Protect(data []byte) ([]byte, error)
	Unprotect(data []byte) ([]byte, error)
}

type SecretStore interface {
	Save(name, value string) error
	Load(name string) (string, error)
}

type FileSecretStore struct {
	rootDir   string
	protector SecretProtector
}

func NewFileSecretStore(rootDir string, protector SecretProtector) FileSecretStore {
	if protector == nil {
		protector = newDefaultSecretProtector()
	}
	return FileSecretStore{
		rootDir:   rootDir,
		protector: protector,
	}
}

func (s FileSecretStore) Save(name, value string) error {
	if name == "" {
		return errors.New("secret name is empty")
	}
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return err
	}

	protected, err := s.protector.Protect([]byte(value))
	if err != nil {
		return err
	}

	return os.WriteFile(s.secretPath(name), protected, 0o600)
}

func (s FileSecretStore) Load(name string) (string, error) {
	data, err := os.ReadFile(s.secretPath(name))
	if err != nil {
		return "", err
	}

	unprotected, err := s.protector.Unprotect(data)
	if err != nil {
		return "", err
	}
	return string(unprotected), nil
}

func (s FileSecretStore) secretPath(name string) string {
	return filepath.Join(s.rootDir, name+".bin")
}
