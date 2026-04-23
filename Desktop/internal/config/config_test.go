package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreSavesAndLoadsRuntimeConfig(t *testing.T) {
	root := t.TempDir()
	paths := DefaultPaths(root)
	store := NewFileStore(paths.ConfigPath)
	cfg := DefaultRuntimeConfig(paths)
	cfg.MediaRoot = filepath.Join(root, "media")
	cfg.SetupComplete = true

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ConfigSchemaVersion != cfg.ConfigSchemaVersion {
		t.Fatalf("ConfigSchemaVersion = %d, want %d", loaded.ConfigSchemaVersion, cfg.ConfigSchemaVersion)
	}
	if loaded.SetupComplete != cfg.SetupComplete {
		t.Fatalf("SetupComplete = %v, want %v", loaded.SetupComplete, cfg.SetupComplete)
	}
	if loaded.MediaRoot != cfg.MediaRoot {
		t.Fatalf("MediaRoot = %q, want %q", loaded.MediaRoot, cfg.MediaRoot)
	}
}

func TestDefaultPathsUsesMilestoneTwoLayout(t *testing.T) {
	root := filepath.Join("C:\\Users\\tester\\AppData\\Local", "NetflixTorrent")

	paths := DefaultPaths(root)

	if paths.ConfigPath != filepath.Join(root, "config", "launcher.json") {
		t.Fatalf("ConfigPath = %q", paths.ConfigPath)
	}
	if paths.BackendEnvPath != filepath.Join(root, "config", "backend.runtime.env") {
		t.Fatalf("BackendEnvPath = %q", paths.BackendEnvPath)
	}
	if paths.PostgresDataDir != filepath.Join(root, "data", "postgres") {
		t.Fatalf("PostgresDataDir = %q", paths.PostgresDataDir)
	}
	if paths.QBittorrentDataDir != filepath.Join(root, "data", "qbittorrent") {
		t.Fatalf("QBittorrentDataDir = %q", paths.QBittorrentDataDir)
	}
	if paths.ProwlarrDataDir != filepath.Join(root, "data", "prowlarr") {
		t.Fatalf("ProwlarrDataDir = %q", paths.ProwlarrDataDir)
	}
	if paths.JackettDataDir != filepath.Join(root, "data", "jackett") {
		t.Fatalf("JackettDataDir = %q", paths.JackettDataDir)
	}
	if paths.DefaultMediaDir != filepath.Join(root, "media", "Movies") {
		t.Fatalf("DefaultMediaDir = %q", paths.DefaultMediaDir)
	}
	if paths.LockFilePath != filepath.Join(root, "run", "netflixtorrent.lock") {
		t.Fatalf("LockFilePath = %q", paths.LockFilePath)
	}
}

func TestMemoryStoreReturnsNotExistWhenEmpty(t *testing.T) {
	var store MemoryStore

	_, err := store.Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want %v", err, os.ErrNotExist)
	}
}
