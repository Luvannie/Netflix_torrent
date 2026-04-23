package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/netflixtorrent/desktop/internal/bootstrap"
	"github.com/netflixtorrent/desktop/internal/config"
	"github.com/netflixtorrent/desktop/internal/processes"
)

type appFakeRunner struct {
	failFor  string
	started  []string
	stopped  []string
	env      map[string]map[string]string
	services map[string]processes.Service
}

type appFakeHealthChecker struct {
	results []error
	calls   int
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
	if r.env == nil {
		r.env = map[string]map[string]string{}
	}
	if r.services == nil {
		r.services = map[string]processes.Service{}
	}
	r.env[service.Name] = service.Environment
	r.services[service.Name] = service
	if service.Name == r.failFor {
		return nil, errors.New("boom")
	}
	return &appFakeHandle{name: service.Name, stopped: &r.stopped}, nil
}

func (r *appFakeRunner) serviceFor(name string) processes.Service {
	if r.services == nil {
		return processes.Service{}
	}
	return r.services[name]
}

func (r *appFakeRunner) envFor(name string) map[string]string {
	if r.env == nil {
		return nil
	}
	return r.env[name]
}

func (h *appFakeHealthChecker) WaitForHealthy(context.Context, string, time.Duration) error {
	h.calls++
	if len(h.results) == 0 {
		return nil
	}

	result := h.results[0]
	h.results = h.results[1:]
	return result
}

func TestAppStartRuntimeTransitionsToReady(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "secret"
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}
	health := &appFakeHealthChecker{}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  health,
		StartupTimeout: 2 * time.Second,
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
	if health.calls != 1 {
		t.Fatalf("health calls = %d", health.calls)
	}

	diag := app.GetDiagnostics()
	if diag.Status != "ready" {
		t.Fatalf("diagnostics status = %q", diag.Status)
	}
	if got := app.GetProxyConfig().LocalToken; got != "secret" {
		t.Fatalf("proxy LocalToken = %q", got)
	}
}

func TestAppStartRuntimeStopsAtSetupRequiredWhenSetupIncomplete(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "secret"
	cfg.SetupComplete = false
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}
	health := &appFakeHealthChecker{}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  health,
		StartupTimeout: time.Second,
	})

	if err := app.StartRuntime(context.Background()); err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}

	state := app.GetBootstrapState()
	if state.Step != bootstrap.StepSetupRequired {
		t.Fatalf("state.Step = %q", state.Step)
	}
	if len(runner.started) != 0 {
		t.Fatalf("started = %#v, want none", runner.started)
	}
	if health.calls != 0 {
		t.Fatalf("health calls = %d, want 0", health.calls)
	}
}

func TestAppStartRuntimePassesDownloadSavePathToBackendEnvironment(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "secret"
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	cfg.DownloadDefaultSavePath = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  &appFakeHealthChecker{},
		StartupTimeout: time.Second,
	})

	if err := app.StartRuntime(context.Background()); err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}
	defer func() {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	if got := runner.envFor("backend")["DOWNLOAD_DEFAULT_SAVE_PATH"]; got != paths.DefaultMediaDir {
		t.Fatalf("DOWNLOAD_DEFAULT_SAVE_PATH = %q, want %q", got, paths.DefaultMediaDir)
	}
}

func TestAppStartRuntimeBuildsSidecarBootstrapContracts(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "secret"
	cfg.DatabasePassword = "db-secret"
	cfg.QBittorrentPassword = "qb-secret"
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	cfg.DownloadDefaultSavePath = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  &appFakeHealthChecker{},
		StartupTimeout: time.Second,
	})

	if err := app.StartRuntime(context.Background()); err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}
	defer func() {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	postgres := runner.serviceFor("postgres")
	if !reflect.DeepEqual(postgres.Args, []string{"-D", paths.PostgresDataDir, "-h", "127.0.0.1", "-p", "15432"}) {
		t.Fatalf("postgres args = %#v", postgres.Args)
	}

	qbittorrentConfig := filepath.Join(paths.QBittorrentDataDir, "qBittorrent", "config", "qBittorrent.conf")
	assertFileContains(t, qbittorrentConfig, "WebUI\\Address=127.0.0.1")
	assertFileContains(t, qbittorrentConfig, "WebUI\\Port=18082")
	assertFileContains(t, qbittorrentConfig, "Downloads\\SavePath="+paths.DefaultMediaDir)
	qbittorrent := runner.serviceFor("qbittorrent")
	if !containsArg(qbittorrent.Args, "--profile="+paths.QBittorrentDataDir) {
		t.Fatalf("qbittorrent args = %#v", qbittorrent.Args)
	}

	providerConfig := filepath.Join(paths.ProwlarrDataDir, "config.xml")
	assertFileContains(t, providerConfig, "<BindAddress>127.0.0.1</BindAddress>")
	assertFileContains(t, providerConfig, "<Port>19696</Port>")
	provider := runner.serviceFor("provider")
	if !containsArg(provider.Args, "-data="+paths.ProwlarrDataDir) {
		t.Fatalf("provider args = %#v", provider.Args)
	}
}

func TestAppStartRuntimePassesCompleteBackendDesktopEnvironment(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.LocalToken = "local-token"
	cfg.DatabasePassword = "db-secret"
	cfg.QBittorrentPassword = "qb-secret"
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	cfg.DownloadDefaultSavePath = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  &appFakeHealthChecker{},
		StartupTimeout: time.Second,
	})

	if err := app.StartRuntime(context.Background()); err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}
	defer func() {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	env := runner.envFor("backend")
	want := map[string]string{
		"DB_URL":                     "jdbc:postgresql://127.0.0.1:15432/netflixtorrent",
		"DB_USERNAME":                "netflixtorrent",
		"DB_PASSWORD":                "db-secret",
		"QBITTORRENT_URL":            "http://127.0.0.1:18082",
		"QBITTORRENT_USERNAME":       "admin",
		"QBITTORRENT_PASSWORD":       "qb-secret",
		"PROWLARR_URL":               "http://127.0.0.1:19696",
		"JACKETT_URL":                "http://127.0.0.1:19117",
		"TORRENT_SEARCH_PROVIDER":    "prowlarr",
		"APP_LOCAL_TOKEN":            "local-token",
		"DOWNLOAD_DEFAULT_SAVE_PATH": paths.DefaultMediaDir,
	}
	for key, expected := range want {
		if got := env[key]; got != expected {
			t.Fatalf("%s = %q, want %q", key, got, expected)
		}
	}
}

func TestAppStartRuntimeFailsWhenServiceFails(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{failFor: "backend"}
	health := &appFakeHealthChecker{}
	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  health,
		StartupTimeout: time.Second,
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
	if health.calls != 0 {
		t.Fatalf("health calls = %d", health.calls)
	}
}

func TestAppStartRuntimeMarksFailedSidecarInDiagnostics(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{failFor: "qbittorrent"}
	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  &appFakeHealthChecker{},
		StartupTimeout: time.Second,
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
	status := diag.Components["qbittorrent"]
	if status.Status != "DOWN" {
		t.Fatalf("qbittorrent diagnostics = %#v", status)
	}
	if !strings.Contains(status.Message, "Failed to start services") {
		t.Fatalf("qbittorrent diagnostics message = %q", status.Message)
	}
}

func TestAppStartRuntimeFailsWhenBackendHealthCheckFails(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	cfg := config.DefaultRuntimeConfig(paths)
	cfg.SetupComplete = true
	cfg.MediaRoot = paths.DefaultMediaDir
	store := &config.MemoryStore{Config: cfg, Loaded: true}
	runner := &appFakeRunner{}
	health := &appFakeHealthChecker{
		results: []error{errors.New("backend health timeout")},
	}

	app := NewApp(Dependencies{
		ConfigStore:    store,
		Paths:          paths,
		ProcessRunner:  runner,
		HealthChecker:  health,
		StartupTimeout: time.Second,
	})

	err := app.StartRuntime(context.Background())
	if err == nil {
		t.Fatalf("expected StartRuntime() error")
	}

	state := app.GetBootstrapState()
	if state.Step != bootstrap.StepFailed {
		t.Fatalf("state.Step = %q", state.Step)
	}
	if health.calls != 1 {
		t.Fatalf("health calls = %d", health.calls)
	}

	diag := app.GetDiagnostics()
	backendStatus, ok := diag.Components["backend"]
	if !ok {
		t.Fatalf("expected backend diagnostics component")
	}
	if backendStatus.Status != "DOWN" {
		t.Fatalf("backend diagnostics status = %q", backendStatus.Status)
	}

	processStates := app.GetProcessStates()
	wantStates := []processes.ProcessState{
		{Name: "postgres", Status: "stopped"},
		{Name: "qbittorrent", Status: "stopped"},
		{Name: "provider", Status: "stopped"},
		{Name: "backend", Status: "stopped"},
	}
	if !reflect.DeepEqual(processStates, wantStates) {
		t.Fatalf("process states = %#v, want %#v", processStates, wantStates)
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

func assertFileContains(t *testing.T, path string, text string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if !strings.Contains(string(data), text) {
		t.Fatalf("%q does not contain %q; content:\n%s", path, text, string(data))
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
