package bridge

type RuntimeBridge struct {
	AppVersion string `json:"appVersion"`
}

type BootstrapState struct {
	Step         string `json:"step"`
	Message      string `json:"message"`
	BackendURL   string `json:"backendUrl"`
	WebSocketURL string `json:"webSocketUrl"`
}
