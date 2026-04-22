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
	cfg.LocalToken = "secret-token"
	cfg.MediaRoot = filepath.Join(root, "media")

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.LocalToken != cfg.LocalToken {
		t.Fatalf("LocalToken = %q, want %q", loaded.LocalToken, cfg.LocalToken)
	}
	if loaded.MediaRoot != cfg.MediaRoot {
		t.Fatalf("MediaRoot = %q, want %q", loaded.MediaRoot, cfg.MediaRoot)
	}
}

func TestMemoryStoreReturnsNotExistWhenEmpty(t *testing.T) {
	var store MemoryStore

	_, err := store.Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want %v", err, os.ErrNotExist)
	}
}
