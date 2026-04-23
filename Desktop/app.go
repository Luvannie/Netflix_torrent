package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/netflixtorrent/desktop/internal/bootstrap"
	"github.com/netflixtorrent/desktop/internal/bridge"
	"github.com/netflixtorrent/desktop/internal/config"
	"github.com/netflixtorrent/desktop/internal/diagnostics"
	"github.com/netflixtorrent/desktop/internal/processes"
	"github.com/netflixtorrent/desktop/internal/proxy"
)

type Dependencies struct {
	ConfigStore     config.Store
	SecretStore     config.SecretStore
	Paths           config.Paths
	ProcessRunner   processes.Runner
	HealthChecker   bootstrap.HealthChecker
	StartupTimeout  time.Duration
	Diagnostics     *diagnostics.Collector
	DirectoryPicker func() (string, error)
	OpenPath        func(string) error
	Exit            func(int)
}

type App struct {
	mu             sync.RWMutex
	state          bootstrap.State
	cfg            config.RuntimeConfig
	configStore    config.Store
	lock           *bootstrap.InstanceLock
	bridge         bridge.RuntimeBridge
	diagnostics    *diagnostics.Collector
	processes      *processes.Manager
	openPath       func(string) error
	pickDir        func() (string, error)
	exit           func(int)
	proxy          proxy.Config
	health         bootstrap.HealthChecker
	startupTimeout time.Duration
	secrets        config.SecretStore
}

func NewApp(deps Dependencies) *App {
	paths := deps.Paths
	if paths.RootDir == "" {
		paths = config.DefaultPaths(".netflixtorrent")
	} else if paths.ConfigPath == "" {
		paths = config.DefaultPaths(paths.RootDir)
	}

	store := deps.ConfigStore
	if store == nil {
		store = config.NewFileStore(paths.ConfigPath)
	}

	secretStore := deps.SecretStore
	if secretStore == nil {
		defaultSecretStore := config.NewFileSecretStore(paths.SecretsDir, nil)
		secretStore = defaultSecretStore
	}

	cfg, err := store.Load()
	if err != nil {
		cfg = config.DefaultRuntimeConfig(paths)
	}
	cfg.Paths = paths
	if cfg, err = bootstrap.PrepareFirstRun(cfg, secretStore, nil); err != nil {
		cfg = config.DefaultRuntimeConfig(paths)
		collector := diagnostics.NewCollector(nil)
		collector.MarkComponent("first-run", "DOWN", fmt.Sprintf("Failed to prepare first run: %v", err))
		collector.MarkOverall("failed", fmt.Sprintf("Failed to prepare first run: %v", err))
		return &App{
			state: bootstrap.State{
				Step:         bootstrap.StepFailed,
				Message:      fmt.Sprintf("Failed to prepare first run: %v", err),
				BackendURL:   cfg.BackendBaseURL,
				WebSocketURL: cfg.WebSocketURL,
			},
			cfg:            cfg,
			configStore:    store,
			diagnostics:    collector,
			processes:      processes.NewManager(deps.ProcessRunner),
			health:         deps.HealthChecker,
			startupTimeout: deps.StartupTimeout,
			secrets:        secretStore,
			openPath:       deps.OpenPath,
			pickDir:        deps.DirectoryPicker,
			exit:           deps.Exit,
		}
	}

	collector := deps.Diagnostics
	if collector == nil {
		collector = diagnostics.NewCollector(nil)
	}

	healthChecker := deps.HealthChecker
	if healthChecker == nil {
		healthChecker = bootstrap.HTTPHealthChecker{}
	}

	startupTimeout := deps.StartupTimeout
	if startupTimeout <= 0 {
		startupTimeout = 30 * time.Second
	}

	manager := processes.NewManager(deps.ProcessRunner)
	if deps.ProcessRunner == nil {
		execRunner := processes.NewExecRunner(5 * time.Second)
		manager = processes.NewManager(execRunner)
	}
	nativeBridge := bridge.NewNativeBridge(bridge.ExecCommandRunner{})
	app := &App{
		state: bootstrap.State{
			Step:         bootstrap.StepIdle,
			Message:      "Desktop runtime initialized",
			BackendURL:   cfg.BackendBaseURL,
			WebSocketURL: cfg.WebSocketURL,
		},
		cfg:         cfg,
		configStore: store,
		bridge: bridge.RuntimeBridge{
			AppVersion: "0.1.0",
		},
		diagnostics:    collector,
		processes:      manager,
		health:         healthChecker,
		startupTimeout: startupTimeout,
		secrets:        secretStore,
		openPath:       deps.OpenPath,
		pickDir:        deps.DirectoryPicker,
		exit:           deps.Exit,
		proxy: proxy.Config{
			TargetBaseURL: cfg.BackendBaseURL,
			LocalToken:    cfg.LocalToken,
		},
	}
	if app.openPath == nil {
		app.openPath = nativeBridge.OpenPath
	}
	if app.pickDir == nil {
		app.pickDir = nativeBridge.ChooseDirectory
	}
	if app.exit == nil {
		app.exit = os.Exit
	}
	if app.cfg.LocalToken != "" && app.cfg.LocalToken != "replace-me" {
		if err := app.persistConfig(); err != nil {
			app.diagnostics.MarkComponent("secrets", "DOWN", fmt.Sprintf("Failed to persist protected local token: %v", err))
		}
	}
	app.diagnostics.MarkOverall("idle", "Desktop runtime initialized")
	return app
}

func (a *App) StartRuntime(ctx context.Context) error {
	if !a.setupComplete() {
		a.setState(bootstrap.StepSetupRequired, "First-run setup is required")
		a.diagnostics.MarkOverall("setup_required", "First-run setup is required")
		return nil
	}

	a.setState(bootstrap.StepAcquiringLock, "Acquiring desktop instance lock")

	lock, err := bootstrap.AcquireInstanceLock(a.cfg.Paths.LockFilePath)
	if err != nil {
		a.fail("lock", fmt.Sprintf("Failed to acquire instance lock: %v", err))
		return err
	}
	a.lock = lock

	a.setState(bootstrap.StepStartingServices, "Starting local services")
	services, err := bootstrap.PrepareSidecarServices(a.cfg)
	if err != nil {
		a.fail("bootstrap", fmt.Sprintf("Failed to prepare sidecar configs: %v", err))
		return err
	}
	if err := a.processes.StartAll(ctx, services); err != nil {
		a.fail(a.failedServiceName(), fmt.Sprintf("Failed to start services: %v", err))
		return err
	}

	for _, service := range a.processes.Snapshot() {
		a.diagnostics.MarkComponent(service.Name, "UP", "Service started")
	}

	a.setState(bootstrap.StepWaitingHealth, "Waiting for backend health checks")
	a.diagnostics.MarkOverall("starting", "Waiting for backend health checks")
	if err := a.health.WaitForHealthy(ctx, a.cfg.BackendBaseURL, a.startupTimeout); err != nil {
		_ = a.processes.StopAll(ctx)
		a.fail("backend", fmt.Sprintf("Backend health check failed: %v", err))
		return err
	}

	a.setState(bootstrap.StepReady, "Desktop runtime is ready")
	a.diagnostics.MarkOverall("ready", "Desktop runtime is ready")
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	var errs []error
	if err := a.processes.StopAll(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := a.lock.Release(); err != nil {
		errs = append(errs, err)
	}

	a.mu.Lock()
	a.lock = nil
	a.state.Step = bootstrap.StepIdle
	a.state.Message = "Desktop runtime stopped"
	a.mu.Unlock()
	a.diagnostics.MarkOverall("stopped", "Desktop runtime stopped")
	return errors.Join(errs...)
}

func (a *App) GetBootstrapState() bridge.BootstrapState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

func (a *App) GetDiagnostics() diagnostics.Snapshot {
	return a.diagnostics.Snapshot()
}

func (a *App) GetProcessStates() []processes.ProcessState {
	return a.processes.Snapshot()
}

func (a *App) GetProxyConfig() proxy.Config {
	return a.proxy
}

func (a *App) ChooseDirectory() (string, error) {
	if a.pickDir == nil {
		return "", errors.New("directory picker is not configured")
	}
	return a.pickDir()
}

func (a *App) SaveLauncherConfig(input config.LauncherSettings) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if input.MediaRoot != "" {
		a.cfg.MediaRoot = input.MediaRoot
		a.cfg.DownloadDefaultSavePath = input.MediaRoot
		a.cfg.SetupComplete = true
	}
	if err := a.persistConfig(); err != nil {
		return err
	}
	return nil
}

func (a *App) RestartBackend(ctx context.Context) error {
	if err := a.processes.Restart(ctx, "backend"); err != nil {
		return err
	}
	a.diagnostics.MarkComponent("backend", "UP", "Service restarted")
	return nil
}

func (a *App) RestartSidecar(ctx context.Context, name string) error {
	if name == "backend" {
		return a.RestartBackend(ctx)
	}
	if err := a.processes.Restart(ctx, name); err != nil {
		return err
	}
	a.diagnostics.MarkComponent(name, "UP", "Service restarted")
	return nil
}

func (a *App) OpenLogsFolder() error {
	if a.openPath == nil {
		return errors.New("path opener is not configured")
	}
	return a.openPath(a.cfg.Paths.LogsDir)
}

func (a *App) GetLogBundlePath() string {
	return a.cfg.Paths.LogsDir + "\\support-bundle.zip"
}

func (a *App) QuitApp() {
	if a.exit != nil {
		a.exit(0)
	}
}

func (a *App) setState(step bootstrap.Step, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state.Step = step
	a.state.Message = message
	a.state.BackendURL = a.cfg.BackendBaseURL
	a.state.WebSocketURL = a.cfg.WebSocketURL
}

func (a *App) fail(component, message string) {
	if a.lock != nil {
		_ = a.lock.Release()
		a.lock = nil
	}
	a.setState(bootstrap.StepFailed, message)
	a.diagnostics.MarkComponent(component, "DOWN", message)
	a.diagnostics.MarkOverall("failed", message)
}

func (a *App) persistConfig() error {
	if a.secrets != nil {
		for name, value := range map[string]string{
			"local-token":          a.cfg.LocalToken,
			"database-password":    a.cfg.DatabasePassword,
			"qbittorrent-password": a.cfg.QBittorrentPassword,
		} {
			if value == "" || value == "replace-me" {
				continue
			}
			if err := a.secrets.Save(name, value); err != nil {
				return err
			}
		}
	}
	return a.configStore.Save(bootstrap.SanitizeForSave(a.cfg))
}

func (a *App) setupComplete() bool {
	return a.cfg.SetupComplete && a.cfg.MediaRoot != "" && a.cfg.DownloadDefaultSavePath != ""
}

func (a *App) backendEnvironment() map[string]string {
	return bootstrap.BackendEnvironment(a.cfg)
}

func (a *App) failedServiceName() string {
	for _, service := range a.processes.Snapshot() {
		if service.Status == "failed" {
			return service.Name
		}
	}
	return "startup"
}
