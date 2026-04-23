package bootstrap

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"

	"github.com/netflixtorrent/desktop/internal/config"
)

const (
	secretLocalToken          = "local-token"
	secretDatabasePassword    = "database-password"
	secretQBittorrentPassword = "qbittorrent-password"
)

type SecretGenerator func() (string, error)

func PrepareFirstRun(cfg config.RuntimeConfig, secrets config.SecretStore, generator SecretGenerator) (config.RuntimeConfig, error) {
	if secrets == nil {
		return cfg, errors.New("secret store is not configured")
	}
	if generator == nil {
		generator = GenerateSecret
	}

	cfg.Paths = fillMissingPaths(cfg.Paths)
	if cfg.ConfigSchemaVersion == 0 {
		cfg.ConfigSchemaVersion = config.CurrentSchemaVersion
	}
	if cfg.DownloadDefaultSavePath == "" {
		cfg.DownloadDefaultSavePath = cfg.Paths.DefaultMediaDir
	}

	if err := ensureRuntimeDirectories(cfg.Paths); err != nil {
		return cfg, err
	}

	var err error
	if cfg.LocalToken, err = loadOrCreateSecret(secrets, secretLocalToken, cfg.LocalToken, generator); err != nil {
		return cfg, err
	}
	if cfg.DatabasePassword, err = loadOrCreateSecret(secrets, secretDatabasePassword, cfg.DatabasePassword, generator); err != nil {
		return cfg, err
	}
	if cfg.QBittorrentPassword, err = loadOrCreateSecret(secrets, secretQBittorrentPassword, cfg.QBittorrentPassword, generator); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func SanitizeForSave(cfg config.RuntimeConfig) config.RuntimeConfig {
	cfg.LocalToken = ""
	cfg.DatabasePassword = ""
	cfg.QBittorrentPassword = ""
	return cfg
}

func GenerateSecret() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func ensureRuntimeDirectories(paths config.Paths) error {
	dirs := []string{
		paths.ConfigDir,
		paths.LogsDir,
		paths.DataDir,
		paths.PostgresDataDir,
		paths.QBittorrentDataDir,
		paths.ProwlarrDataDir,
		paths.JackettDataDir,
		paths.SecretsDir,
		paths.DefaultMediaDir,
		paths.RunDir,
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func fillMissingPaths(paths config.Paths) config.Paths {
	if paths.RootDir == "" {
		return config.DefaultPaths(filepath.Join(".", ".netflixtorrent"))
	}
	defaults := config.DefaultPaths(paths.RootDir)
	if paths.ConfigDir == "" {
		paths.ConfigDir = defaults.ConfigDir
	}
	if paths.ConfigPath == "" {
		paths.ConfigPath = defaults.ConfigPath
	}
	if paths.BackendEnvPath == "" {
		paths.BackendEnvPath = defaults.BackendEnvPath
	}
	if paths.LogsDir == "" {
		paths.LogsDir = defaults.LogsDir
	}
	if paths.DataDir == "" {
		paths.DataDir = defaults.DataDir
	}
	if paths.PostgresDataDir == "" {
		paths.PostgresDataDir = defaults.PostgresDataDir
	}
	if paths.QBittorrentDataDir == "" {
		paths.QBittorrentDataDir = defaults.QBittorrentDataDir
	}
	if paths.ProwlarrDataDir == "" {
		paths.ProwlarrDataDir = defaults.ProwlarrDataDir
	}
	if paths.JackettDataDir == "" {
		paths.JackettDataDir = defaults.JackettDataDir
	}
	if paths.SecretsDir == "" {
		paths.SecretsDir = defaults.SecretsDir
	}
	if paths.DefaultMediaDir == "" {
		paths.DefaultMediaDir = defaults.DefaultMediaDir
	}
	if paths.RunDir == "" {
		paths.RunDir = defaults.RunDir
	}
	if paths.LockFilePath == "" {
		paths.LockFilePath = defaults.LockFilePath
	}
	return paths
}

func loadOrCreateSecret(store config.SecretStore, name, fallback string, generator SecretGenerator) (string, error) {
	current, err := store.Load(name)
	if err == nil && current != "" {
		return current, nil
	}
	if fallback != "" && fallback != "replace-me" {
		if err := store.Save(name, fallback); err != nil {
			return "", err
		}
		return fallback, nil
	}
	generated, err := generator()
	if err != nil {
		return "", err
	}
	if err := store.Save(name, generated); err != nil {
		return "", err
	}
	return generated, nil
}
