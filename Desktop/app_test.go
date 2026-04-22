package main

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/netflixtorrent/desktop/internal/bootstrap"
	"github.com/netflixtorrent/desktop/internal/config"
	"github.com/netflixtorrent/desktop/internal/processes"
)

type appFakeRunner struct {
	failFor string
	started []string
	stopped []string
}

type appFakeHandle struct {
	name    string
	stopped *[]string
}

func (h *appFakeHandle) Stop(context.Context) error {
	*h.stopped = append(*h.stopped, h.name)
	return nil
}

func (r *appFakeRunner) Start(_ context.Context, service processes.Service) (processes.Handle, error) {
	r.started = append(r.started, service.Name)
	if service.Name == r.failFor {
		return nil, errors.New("boom")
	}
	return &appFakeHandle{name: service.Name, stopped: &r.stopped}, nil
}

func TestAppStartRuntimeTransitionsToReady(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "secret"
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}

	app := NewApp(Dependencies{
		ConfigStore:   store,
		Paths:         paths,
		ProcessRunner: runner,
	})

	if err := app.StartRuntime(context.Background()); err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}
	defer func() {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	state := app.GetBootstrapState()
	if state.Step != bootstrap.StepReady {
		t.Fatalf("state.Step = %q", state.Step)
	}
	if !reflect.DeepEqual(runner.started, []string{"postgres", "qbittorrent", "provider", "backend"}) {
		t.Fatalf("started = %#v", runner.started)
	}

	diag := app.GetDiagnostics()
	if diag.Status != "ready" {
		t.Fatalf("diagnostics status = %q", diag.Status)
	}
	if got := app.GetProxyConfig().LocalToken; got != "secret" {
		t.Fatalf("proxy LocalToken = %q", got)
	}
}

func TestAppStartRuntimeFailsWhenServiceFails(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	store := &config.MemoryStore{Config: config.DefaultRuntimeConfig(paths), Loaded: true}
	runner := &appFakeRunner{failFor: "backend"}
	app := NewApp(Dependencies{
		ConfigStore:   store,
		Paths:         paths,
		ProcessRunner: runner,
	})

	err := app.StartRuntime(context.Background())
	if err == nil {
		t.Fatalf("expected StartRuntime() error")
	}

	state := app.GetBootstrapState()
	if state.Step != bootstrap.StepFailed {
		t.Fatalf("state.Step = %q", state.Step)
	}
	diag := app.GetDiagnostics()
	if diag.Status != "failed" {
		t.Fatalf("diagnostics status = %q", diag.Status)
	}
}

func TestAppSaveLauncherConfigPersistsMediaRoot(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	store := config.NewFileStore(paths.ConfigPath)
	app := NewApp(Dependencies{
		ConfigStore: &store,
		Paths:       paths,
	})

	mediaRoot := filepath.Join(root, "media")
	if err := app.SaveLauncherConfig(config.LauncherSettings{MediaRoot: mediaRoot}); err != nil {
		t.Fatalf("SaveLauncherConfig() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.MediaRoot != mediaRoot {
		t.Fatalf("MediaRoot = %q, want %q", loaded.MediaRoot, mediaRoot)
	}
}

func TestAppBridgeMethodsDelegateToInjectedFunctions(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	store := &config.MemoryStore{Config: config.DefaultRuntimeConfig(paths), Loaded: true}
	var openedPath string
	var exitCode int = -1

	app := NewApp(Dependencies{
		ConfigStore: store,
		Paths:       paths,
		DirectoryPicker: func() (string, error) {
			return filepath.Join(root, "picked"), nil
		},
		OpenPath: func(path string) error {
			openedPath = path
			return nil
		},
		Exit: func(code int) {
			exitCode = code
		},
	})

	dir, err := app.ChooseDirectory()
	if err != nil {
		t.Fatalf("ChooseDirectory() error = %v", err)
	}
	if dir != filepath.Join(root, "picked") {
		t.Fatalf("ChooseDirectory() = %q", dir)
	}

	if err := app.OpenLogsFolder(); err != nil {
		t.Fatalf("OpenLogsFolder() error = %v", err)
	}
	if openedPath != paths.LogsDir {
		t.Fatalf("OpenLogsFolder opened %q, want %q", openedPath, paths.LogsDir)
	}

	app.QuitApp()
	if exitCode != 0 {
		t.Fatalf("exitCode = %d", exitCode)
	}
}
