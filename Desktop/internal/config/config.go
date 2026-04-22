package config

type RuntimeConfig struct {
	BackendBaseURL string `json:"backendBaseUrl"`
	WebSocketURL   string `json:"webSocketUrl"`
	LocalToken     string `json:"localToken"`
	MediaRoot      string `json:"mediaRoot"`
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		BackendBaseURL: "http://127.0.0.1:18080",
		WebSocketURL:   "ws://127.0.0.1:18080/ws",
		LocalToken:     "replace-me",
		MediaRoot:      "",
	}
}
