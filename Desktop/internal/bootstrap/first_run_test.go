package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netflixtorrent/desktop/internal/config"
)

type memorySecretStore struct {
	values map[string]string
}

func newMemorySecretStore() *memorySecretStore {
	return &memorySecretStore{values: map[string]string{}}
}

func (s *memorySecretStore) Save(name, value string) error {
	s.values[name] = value
	return nil
}

func (s *memorySecretStore) Load(name string) (string, error) {
	value, ok := s.values[name]
	if !ok {
		return "", os.ErrNotExist
	}
	return value, nil
}

func TestPrepareFirstRunCreatesRuntimeLayoutAndSecrets(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	secrets := newMemorySecretStore()
	nextSecret := 0

	cfg, err := PrepareFirstRun(config.DefaultRuntimeConfig(paths), secrets, func() (string, error) {
		nextSecret++
		return []string{"local-token-value", "db-password-value", "qb-password-value"}[nextSecret-1], nil
	})
	if err != nil {
		t.Fatalf("PrepareFirstRun() error = %v", err)
	}

	for _, dir := range []string{
		filepath.Join(root, "config"),
		paths.LogsDir,
		paths.PostgresDataDir,
		paths.QBittorrentDataDir,
		paths.ProwlarrDataDir,
		paths.JackettDataDir,
		paths.DefaultMediaDir,
		filepath.Join(root, "run"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %q: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}

	if cfg.LocalToken != "local-token-value" {
		t.Fatalf("LocalToken = %q", cfg.LocalToken)
	}
	if cfg.DatabasePassword != "db-password-value" {
		t.Fatalf("DatabasePassword = %q", cfg.DatabasePassword)
	}
	if cfg.QBittorrentPassword != "qb-password-value" {
		t.Fatalf("QBittorrentPassword = %q", cfg.QBittorrentPassword)
	}
	if cfg.DownloadDefaultSavePath != paths.DefaultMediaDir {
		t.Fatalf("DownloadDefaultSavePath = %q, want %q", cfg.DownloadDefaultSavePath, paths.DefaultMediaDir)
	}
}

func TestPrepareFirstRunReusesExistingSecrets(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	secrets := newMemorySecretStore()
	secrets.values["local-token"] = "existing-token"
	secrets.values["database-password"] = "existing-db"
	secrets.values["qbittorrent-password"] = "existing-qb"

	cfg, err := PrepareFirstRun(config.DefaultRuntimeConfig(paths), secrets, func() (string, error) {
		t.Fatalf("secret generator should not be called")
		return "", nil
	})
	if err != nil {
		t.Fatalf("PrepareFirstRun() error = %v", err)
	}

	if cfg.LocalToken != "existing-token" || cfg.DatabasePassword != "existing-db" || cfg.QBittorrentPassword != "existing-qb" {
		t.Fatalf("cfg secrets = %#v", cfg)
	}
}

func TestSanitizeForSaveRemovesPlaintextSecrets(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultRuntimeConfig(config.DefaultPaths(root))
	cfg.LocalToken = "local-token-value"
	cfg.DatabasePassword = "db-password-value"
	cfg.QBittorrentPassword = "qb-password-value"

	sanitized := SanitizeForSave(cfg)
	if sanitized.LocalToken != "" || sanitized.DatabasePassword != "" || sanitized.QBittorrentPassword != "" {
		t.Fatalf("expected sanitized config to remove plaintext secrets: %#v", sanitized)
	}

	data, err := config.MarshalRuntimeConfig(sanitized)
	if err != nil {
		t.Fatalf("MarshalRuntimeConfig() error = %v", err)
	}
	if text := string(data); strings.Contains(text, "local-token-value") || strings.Contains(text, "db-password-value") || strings.Contains(text, "qb-password-value") {
		t.Fatalf("sanitized config contains plaintext secret: %s", text)
	}
}
