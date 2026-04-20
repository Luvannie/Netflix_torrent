package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteOKUsesEnvelopeShape(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteOK(rec, http.StatusOK, map[string]string{"status": "UP"}, "req-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["data"] == nil {
		t.Fatalf("missing data: %s", rec.Body.String())
	}
	meta := decoded["meta"].(map[string]any)
	if meta["timestamp"] == "" {
		t.Fatalf("missing timestamp: %s", rec.Body.String())
	}
	if meta["requestId"] != "req-1" {
		t.Fatalf("requestId = %v", meta["requestId"])
	}
}

func TestWriteOKWritesNullRequestIDWhenHeaderMissing(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteOK(rec, http.StatusOK, map[string]string{"status": "UP"}, "")

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	meta := decoded["meta"].(map[string]any)
	if meta["requestId"] != nil {
		t.Fatalf("requestId = %v, want nil", meta["requestId"])
	}
}

func TestWriteErrorUsesErrorEnvelopeShape(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusForbidden, "ACCESS_DENIED", "Access denied.", nil, "req-2")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["error"] == nil {
		t.Fatalf("missing error: %s", rec.Body.String())
	}
	errBody := decoded["error"].(map[string]any)
	if errBody["code"] != "ACCESS_DENIED" {
		t.Fatalf("code = %v", errBody["code"])
	}
	if errBody["message"] != "Access denied." {
		t.Fatalf("message = %v", errBody["message"])
	}
	details := errBody["details"].([]any)
	if len(details) != 0 {
		t.Fatalf("details = %v", details)
	}
}

func TestWriteErrorWithValidationDetails(t *testing.T) {
	rec := httptest.NewRecorder()

	details := []ErrorDetail{
		{Field: "query", Message: "Query must not be blank"},
	}
	WriteError(rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Request validation failed.", details, "")

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errBody := decoded["error"].(map[string]any)
	detailsArr := errBody["details"].([]any)
	if len(detailsArr) != 1 {
		t.Fatalf("details count = %d", len(detailsArr))
	}
	firstDetail := detailsArr[0].(map[string]any)
	if firstDetail["field"] != "query" {
		t.Fatalf("field = %v", firstDetail["field"])
	}
}

func TestMetaTimestampIsRecent(t *testing.T) {
	before := time.Now().UTC()
	meta := NewMeta("test")
	after := time.Now().UTC()

	if meta.Timestamp.Before(before) || meta.Timestamp.After(after) {
		t.Fatalf("Timestamp not in range: %v", meta.Timestamp)
	}
}