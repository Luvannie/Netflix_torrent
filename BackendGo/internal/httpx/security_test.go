package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalOnlyMiddlewareAllowsLoopback(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:51234"
	rec := httptest.NewRecorder()

	LocalOnlyMiddleware(true)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalOnlyMiddlewareBlocksNonLoopback(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "192.168.1.20:51234"
	rec := httptest.NewRecorder()

	LocalOnlyMiddleware(true)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalOnlyMiddlewareDisabledAllowsAll(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "192.168.1.20:51234"
	rec := httptest.NewRecorder()

	LocalOnlyMiddleware(false)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareAllowsReadRequestWithoutToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(true, "secret")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareRequiresTokenForProtectedWritePath(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(true, "secret")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareRejectsInvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/settings/storage-profiles/1", nil)
	req.Header.Set("X-App-Local-Token", "wrong")
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(true, "secret")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareAllowsValidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/jobs", nil)
	req.Header.Set("X-App-Local-Token", "secret")
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(true, "secret")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareReturnsUnavailableWhenTokenIsNotConfigured(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(true, "")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalTokenMiddlewareDisabledAllowsWriteWithoutToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/downloads", nil)
	rec := httptest.NewRecorder()

	LocalTokenMiddleware(false, "secret")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCORSMiddlewareReturns204WithCORSHeadersOnOptions(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := CORSMiddleware([]string{"http://localhost:5173"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/catalog", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods is empty")
	}
}

func TestRequestIDMiddlewareKeepsInboundHeaderForResponseMeta(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := EffectiveRequestID(r); got != "req-1" {
			t.Fatalf("EffectiveRequestID = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-Id", "req-1")
	rec := httptest.NewRecorder()

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("X-Request-Id"); got != "req-1" {
		t.Fatalf("response request id = %q", got)
	}
}

func TestRequestIDMiddlewareGeneratesEffectiveIDWhenHeaderMissing(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := EffectiveRequestID(r); got == "" {
			t.Fatalf("EffectiveRequestID is empty")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("X-Request-Id"); got == "" {
		t.Fatalf("response request id is empty")
	}
}