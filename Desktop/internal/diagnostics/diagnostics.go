package diagnostics

type Snapshot struct {
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}
