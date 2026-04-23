package bootstrap

type Step string

const (
	StepIdle             Step = "IDLE"
	StepAcquiringLock    Step = "ACQUIRING_LOCK"
	StepStartingServices Step = "STARTING_SERVICES"
	StepWaitingHealth    Step = "WAITING_HEALTH"
	StepSetupRequired    Step = "SETUP_REQUIRED"
	StepReady            Step = "READY"
	StepFailed           Step = "FAILED"
)

type State struct {
	Step         Step   `json:"step"`
	Message      string `json:"message"`
	BackendURL   string `json:"backendUrl"`
	WebSocketURL string `json:"webSocketUrl"`
}
