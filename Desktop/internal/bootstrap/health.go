package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HealthChecker interface {
	WaitForHealthy(ctx context.Context, backendBaseURL string, timeout time.Duration) error
}

type HTTPHealthChecker struct {
	Client   *http.Client
	Now      func() time.Time
	Interval time.Duration
}

func (c HTTPHealthChecker) WaitForHealthy(ctx context.Context, backendBaseURL string, timeout time.Duration) error {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}

	interval := c.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	deadline := now().Add(timeout)
	healthURL := strings.TrimRight(backendBaseURL, "/") + "/api/v1/health"

	for {
		if now().After(deadline) {
			return fmt.Errorf("backend health timeout after %s", timeout)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
