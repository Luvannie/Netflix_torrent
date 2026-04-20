package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netflixtorrent/backend-go/internal/httpx"
)

func TestHealthReturnsStatusUp(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-Id", "health-req-1")
	rec := httptest.NewRecorder()

	handler := httpx.RequestIDMiddleware(Handler())
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	data := decoded["data"].(map[string]any)
	if data["status"] != "UP" {
		t.Fatalf("status = %v", data["status"])
	}
	if data["service"] != "backend" {
		t.Fatalf("service = %v", data["service"])
	}

	meta := decoded["meta"].(map[string]any)
	if meta["requestId"] != "health-req-1" {
		t.Fatalf("requestId = %v", meta["requestId"])
	}
}

func TestHealthWithNoRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler := httpx.RequestIDMiddleware(Handler())
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	meta := decoded["meta"].(map[string]any)
	if meta["requestId"] != nil {
		t.Fatalf("requestId should be nil, got %v", meta["requestId"])
	}
}