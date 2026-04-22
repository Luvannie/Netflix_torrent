package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildDirectorAddsLocalTokenForProtectedWrites(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-App-Local-Token"); got != "secret" {
			t.Fatalf("expected local token header, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := NewReverseProxy(Config{
		TargetBaseURL: upstream.URL,
		LocalToken:    "secret",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestBuildDirectorSkipsLocalTokenForReads(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-App-Local-Token"); got != "" {
			t.Fatalf("expected no local token header, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := NewReverseProxy(Config{
		TargetBaseURL: upstream.URL,
		LocalToken:    "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
