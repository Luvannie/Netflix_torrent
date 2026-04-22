package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPHealthCheckerWaitForHealthyReturnsOnHealthyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPHealthChecker{
		Client:   server.Client(),
		Interval: 10 * time.Millisecond,
	}

	if err := checker.WaitForHealthy(context.Background(), server.URL, time.Second); err != nil {
		t.Fatalf("WaitForHealthy() error = %v", err)
	}
}

func TestHTTPHealthCheckerWaitForHealthyTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	checker := HTTPHealthChecker{
		Client:   server.Client(),
		Interval: 10 * time.Millisecond,
	}

	err := checker.WaitForHealthy(context.Background(), server.URL, 50*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
