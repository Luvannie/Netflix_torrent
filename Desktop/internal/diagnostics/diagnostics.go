package diagnostics

import (
	"sync"
	"time"
)

type ComponentStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Snapshot struct {
	Status     string                     `json:"status"`
	Message    string                     `json:"message,omitempty"`
	UpdatedAt  string                     `json:"updatedAt"`
	Components map[string]ComponentStatus `json:"components,omitempty"`
}

type Collector struct {
	mu       sync.RWMutex
	clock    func() time.Time
	snapshot Snapshot
}

func NewCollector(clock func() time.Time) *Collector {
	if clock == nil {
		clock = time.Now
	}

	return &Collector{
		clock: clock,
		snapshot: Snapshot{
			Status:     "idle",
			UpdatedAt:  clock().UTC().Format(time.RFC3339),
			Components: map[string]ComponentStatus{},
		},
	}
}

func (c *Collector) MarkComponent(name, status, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.snapshot.Components[name] = ComponentStatus{
		Status:  status,
		Message: message,
	}
	c.snapshot.UpdatedAt = c.clock().UTC().Format(time.RFC3339)
}

func (c *Collector) MarkOverall(status, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.snapshot.Status = status
	c.snapshot.Message = message
	c.snapshot.UpdatedAt = c.clock().UTC().Format(time.RFC3339)
}

func (c *Collector) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	components := make(map[string]ComponentStatus, len(c.snapshot.Components))
	for name, status := range c.snapshot.Components {
		components[name] = status
	}

	return Snapshot{
		Status:     c.snapshot.Status,
		Message:    c.snapshot.Message,
		UpdatedAt:  c.snapshot.UpdatedAt,
		Components: components,
	}
}
