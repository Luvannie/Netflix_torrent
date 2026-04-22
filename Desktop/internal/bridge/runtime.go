package bridge

import "github.com/netflixtorrent/desktop/internal/bootstrap"

type RuntimeBridge struct {
	AppVersion string `json:"appVersion"`
}

type BootstrapState = bootstrap.State
