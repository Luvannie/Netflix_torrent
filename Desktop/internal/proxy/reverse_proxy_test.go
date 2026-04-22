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

func TestReverseProxyPreservesRangeHeaderForStreams(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/streams/42" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Range"); got != "bytes=0-1023" {
			t.Fatalf("range = %q", got)
		}
		if got := r.Header.Get("X-App-Local-Token"); got != "" {
			t.Fatalf("did not expect token header, got %q", got)
		}
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer upstream.Close()

	proxy := NewReverseProxy(Config{
		TargetBaseURL: upstream.URL,
		LocalToken:    "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/streams/42", nil)
	req.Header.Set("Range", "bytes=0-1023")
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d", rec.Code)
	}
}
