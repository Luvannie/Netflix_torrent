package bootstrap

type Step string

const (
	StepIdle            Step = "IDLE"
	StepStartingBackend Step = "STARTING_BACKEND"
	StepWaitingHealth   Step = "WAITING_HEALTH"
	StepReady           Step = "READY"
	StepFailed          Step = "FAILED"
)

type State struct {
	Step         Step   `json:"step"`
	Message      string `json:"message"`
	BackendURL   string `json:"backendUrl"`
	WebSocketURL string `json:"webSocketUrl"`
}
