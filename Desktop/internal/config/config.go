package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Paths struct {
	RootDir      string `json:"rootDir"`
	ConfigPath   string `json:"configPath"`
	LogsDir      string `json:"logsDir"`
	DataDir      string `json:"dataDir"`
	SecretsDir   string `json:"secretsDir"`
	LockFilePath string `json:"lockFilePath"`
}

func DefaultPaths(root string) Paths {
	return Paths{
		RootDir:      root,
		ConfigPath:   filepath.Join(root, "launcher.json"),
		LogsDir:      filepath.Join(root, "logs"),
		DataDir:      filepath.Join(root, "data"),
		SecretsDir:   filepath.Join(root, "secrets"),
		LockFilePath: filepath.Join(root, "run", "netflixtorrent.lock"),
	}
}

type Executables struct {
	Backend     string `json:"backend"`
	Postgres    string `json:"postgres"`
	QBittorrent string `json:"qBittorrent"`
	Provider    string `json:"provider"`
}

type RuntimeConfig struct {
	BackendBaseURL string      `json:"backendBaseUrl"`
	WebSocketURL   string      `json:"webSocketUrl"`
	LocalToken     string      `json:"localToken"`
	MediaRoot      string      `json:"mediaRoot"`
	Paths          Paths       `json:"paths"`
	Executables    Executables `json:"executables"`
}

type LauncherSettings struct {
	MediaRoot string `json:"mediaRoot"`
}

func DefaultRuntimeConfig(paths Paths) RuntimeConfig {
	return RuntimeConfig{
		BackendBaseURL: "http://127.0.0.1:18080",
		WebSocketURL:   "ws://127.0.0.1:18080/ws",
		LocalToken:     "replace-me",
		MediaRoot:      "",
		Paths:          paths,
		Executables: Executables{
			Backend:     filepath.Join(paths.RootDir, "bin", "backend.exe"),
			Postgres:    filepath.Join(paths.RootDir, "bin", "postgres.exe"),
			QBittorrent: filepath.Join(paths.RootDir, "bin", "qbittorrent.exe"),
			Provider:    filepath.Join(paths.RootDir, "bin", "prowlarr.exe"),
		},
	}
}

type Store interface {
	Load() (RuntimeConfig, error)
	Save(RuntimeConfig) error
}

type FileStore struct {
	Path string
}

func NewFileStore(path string) FileStore {
	return FileStore{Path: path}
}

func (s FileStore) Load() (RuntimeConfig, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return RuntimeConfig{}, err
	}

	var cfg RuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return RuntimeConfig{}, err
	}
	return cfg, nil
}

func (s FileStore) Save(cfg RuntimeConfig) error {
	if s.Path == "" {
		return errors.New("config path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.Path, data, 0o600)
}

type MemoryStore struct {
	Config RuntimeConfig
	Loaded bool
}

func (s *MemoryStore) Load() (RuntimeConfig, error) {
	if !s.Loaded {
		return RuntimeConfig{}, os.ErrNotExist
	}
	return s.Config, nil
}

func (s *MemoryStore) Save(cfg RuntimeConfig) error {
	s.Config = cfg
	s.Loaded = true
	return nil
}
