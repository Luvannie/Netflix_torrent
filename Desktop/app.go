package main

import (
	"github.com/netflixtorrent/desktop/internal/bootstrap"
	"github.com/netflixtorrent/desktop/internal/bridge"
	"github.com/netflixtorrent/desktop/internal/config"
	"github.com/netflixtorrent/desktop/internal/diagnostics"
	"github.com/netflixtorrent/desktop/internal/processes"
	"github.com/netflixtorrent/desktop/internal/proxy"
)

type App struct {
	State       bootstrap.State
	Config      config.RuntimeConfig
	Bridge      bridge.RuntimeBridge
	Diagnostics diagnostics.Snapshot
	Processes   processes.Manager
	Proxy       proxy.Config
}

func NewApp() *App {
	cfg := config.DefaultRuntimeConfig()
	return &App{
		State: bootstrap.State{
			Step:         bootstrap.StepIdle,
			Message:      "Desktop shell initialized",
			BackendURL:   cfg.BackendBaseURL,
			WebSocketURL: cfg.WebSocketURL,
		},
		Config: cfg,
		Bridge: bridge.RuntimeBridge{
			AppVersion: "0.1.0",
		},
		Diagnostics: diagnostics.Snapshot{
			Status: "idle",
		},
		Processes: processes.NewManager(),
		Proxy: proxy.Config{
			TargetBaseURL: cfg.BackendBaseURL,
			LocalToken:    cfg.LocalToken,
		},
	}
}
