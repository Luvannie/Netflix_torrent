package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const CurrentSchemaVersion = 1

type Paths struct {
	RootDir            string `json:"rootDir"`
	ConfigDir          string `json:"configDir"`
	ConfigPath         string `json:"configPath"`
	BackendEnvPath     string `json:"backendEnvPath"`
	LogsDir            string `json:"logsDir"`
	DataDir            string `json:"dataDir"`
	PostgresDataDir    string `json:"postgresDataDir"`
	QBittorrentDataDir string `json:"qBittorrentDataDir"`
	ProwlarrDataDir    string `json:"prowlarrDataDir"`
	JackettDataDir     string `json:"jackettDataDir"`
	SecretsDir         string `json:"secretsDir"`
	DefaultMediaDir    string `json:"defaultMediaDir"`
	RunDir             string `json:"runDir"`
	LockFilePath       string `json:"lockFilePath"`
}

func DefaultPaths(root string) Paths {
	configDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	runDir := filepath.Join(root, "run")
	return Paths{
		RootDir:            root,
		ConfigDir:          configDir,
		ConfigPath:         filepath.Join(configDir, "launcher.json"),
		BackendEnvPath:     filepath.Join(configDir, "backend.runtime.env"),
		LogsDir:            filepath.Join(root, "logs"),
		DataDir:            dataDir,
		PostgresDataDir:    filepath.Join(dataDir, "postgres"),
		QBittorrentDataDir: filepath.Join(dataDir, "qbittorrent"),
		ProwlarrDataDir:    filepath.Join(dataDir, "prowlarr"),
		JackettDataDir:     filepath.Join(dataDir, "jackett"),
		SecretsDir:         filepath.Join(root, "secrets"),
		DefaultMediaDir:    filepath.Join(root, "media", "Movies"),
		RunDir:             runDir,
		LockFilePath:       filepath.Join(runDir, "netflixtorrent.lock"),
	}
}

type Executables struct {
	Backend     string `json:"backend"`
	Postgres    string `json:"postgres"`
	QBittorrent string `json:"qBittorrent"`
	Provider    string `json:"provider"`
}

type Ports struct {
	Backend     int `json:"backend"`
	Postgres    int `json:"postgres"`
	QBittorrent int `json:"qBittorrent"`
	Prowlarr    int `json:"prowlarr"`
	Jackett     int `json:"jackett"`
}

type RuntimeConfig struct {
	ConfigSchemaVersion     int         `json:"configSchemaVersion"`
	SetupComplete           bool        `json:"setupComplete"`
	BackendBaseURL          string      `json:"backendBaseUrl"`
	WebSocketURL            string      `json:"webSocketUrl"`
	LocalToken              string      `json:"-"`
	DatabasePassword        string      `json:"-"`
	QBittorrentPassword     string      `json:"-"`
	MediaRoot               string      `json:"mediaRoot"`
	DownloadDefaultSavePath string      `json:"downloadDefaultSavePath"`
	SearchProvider          string      `json:"searchProvider"`
	Ports                   Ports       `json:"ports"`
	Paths                   Paths       `json:"paths"`
	Executables             Executables `json:"executables"`
}

type LauncherSettings struct {
	MediaRoot string `json:"mediaRoot"`
}

func DefaultRuntimeConfig(paths Paths) RuntimeConfig {
	return RuntimeConfig{
		ConfigSchemaVersion:     CurrentSchemaVersion,
		SetupComplete:           false,
		BackendBaseURL:          "http://127.0.0.1:18080",
		WebSocketURL:            "ws://127.0.0.1:18080/ws",
		MediaRoot:               "",
		DownloadDefaultSavePath: paths.DefaultMediaDir,
		SearchProvider:          "prowlarr",
		Ports: Ports{
			Backend:     18080,
			Postgres:    15432,
			QBittorrent: 18082,
			Prowlarr:    19696,
			Jackett:     19117,
		},
		Paths: paths,
		Executables: Executables{
			Backend:     filepath.Join(paths.RootDir, "bin", "backend.exe"),
			Postgres:    filepath.Join(paths.RootDir, "bin", "postgres.exe"),
			QBittorrent: filepath.Join(paths.RootDir, "bin", "qbittorrent.exe"),
			Provider:    filepath.Join(paths.RootDir, "bin", "prowlarr.exe"),
		},
	}
}

func MarshalRuntimeConfig(cfg RuntimeConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
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

	data, err := MarshalRuntimeConfig(cfg)
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
