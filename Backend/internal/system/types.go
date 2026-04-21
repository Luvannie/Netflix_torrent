package system

import "time"

type ComponentStatus struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type SystemStatusResponse struct {
	OverallStatus  string                     `json:"overallStatus"`
	Mode           string                     `json:"mode"`
	ActiveProfiles []string                   `json:"activeProfiles"`
	Components     map[string]ComponentStatus `json:"components"`
	CheckedAt      time.Time                  `json:"checkedAt"`
}

const (
	StatusUp      = "UP"
	StatusDown    = "DOWN"
	StatusDisabled = "DISABLED"
)