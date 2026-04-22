package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/netflixtorrent/backend-go/internal/httpx"
)

type mockDBPinger struct {
	pingErr error
}

func (m *mockDBPinger) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestSystemStatusContainsAllComponentKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storageRoot := t.TempDir()
	service := Service{
		Mode:           "local",
		ActiveProfiles: []string{"local"},
		DBPing:         &mockDBPinger{},
		StoragePath:    storageRoot,
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		ProwlarrURL:    server.URL,
		SearchProvider: "both",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	handler := httpx.RequestIDMiddleware(Handler(service))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	req.Header.Set("X-Request-Id", "sys-req-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	data := decoded["data"].(map[string]any)
	components := data["components"].(map[string]any)

	for _, key := range []string{"database", "storage", "ffprobe", "qbittorrent", "jackett", "prowlarr"} {
		if _, ok := components[key]; !ok {
			t.Fatalf("missing component key %q", key)
		}
	}
}

func TestSystemStatusOverallIsDownWhenDatabaseIsDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := Service{
		Mode:           "local",
		ActiveProfiles: []string{"local"},
		DBPing:         &mockDBPinger{pingErr: context.DeadlineExceeded},
		StoragePath:    t.TempDir(),
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		SearchProvider: "jackett",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	handler := httpx.RequestIDMiddleware(Handler(service))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var decoded map[string]any
	json.Unmarshal(rec.Body.Bytes(), &decoded)
	data := decoded["data"].(map[string]any)

	if data["overallStatus"] != "DOWN" {
		t.Fatalf("overallStatus = %v, want DOWN", data["overallStatus"])
	}
}

func TestSystemStatusOverallIsUpWhenAllComponentsUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := Service{
		Mode:           "local",
		ActiveProfiles: []string{"local"},
		DBPing:         &mockDBPinger{},
		StoragePath:    t.TempDir(),
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		ProwlarrURL:    server.URL,
		SearchProvider: "both",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	handler := httpx.RequestIDMiddleware(Handler(service))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var decoded map[string]any
	json.Unmarshal(rec.Body.Bytes(), &decoded)
	data := decoded["data"].(map[string]any)

	if data["overallStatus"] != "UP" {
		t.Fatalf("overallStatus = %v, want UP", data["overallStatus"])
	}
}

func TestJackettDisabledWhenProviderIsProwlarr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := Service{
		Mode:           "desktop",
		ActiveProfiles: []string{"desktop"},
		DBPing:         &mockDBPinger{},
		StoragePath:    t.TempDir(),
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		ProwlarrURL:    server.URL,
		SearchProvider: "prowlarr",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	status := service.Status(context.Background())
	jackett := status.Components["jackett"]

	if jackett.Status != StatusDisabled {
		t.Fatalf("jackett status = %v, want DISABLED", jackett.Status)
	}
}

func TestProwlarrDisabledWhenProviderIsJackett(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := Service{
		Mode:           "desktop",
		ActiveProfiles: []string{"desktop"},
		DBPing:         &mockDBPinger{},
		StoragePath:    t.TempDir(),
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		ProwlarrURL:    server.URL,
		SearchProvider: "jackett",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	status := service.Status(context.Background())
	prowlarr := status.Components["prowlarr"]

	if prowlarr.Status != StatusDisabled {
		t.Fatalf("prowlarr status = %v, want DISABLED", prowlarr.Status)
	}
}

func TestBothJackettAndProwlarrUpWhenProviderIsBoth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := Service{
		Mode:           "desktop",
		ActiveProfiles: []string{"desktop", "worker"},
		DBPing:         &mockDBPinger{},
		StoragePath:    t.TempDir(),
		FFProbePath:    "ffprobe",
		QBittorrentURL: server.URL,
		JackettURL:     server.URL,
		ProwlarrURL:    server.URL,
		SearchProvider: "both",
		httpClient:     server.Client(),
		execLookPath: func(command string) (string, error) {
			return command, nil
		},
	}

	status := service.Status(context.Background())

	if status.Components["jackett"].Status != StatusUp {
		t.Fatalf("jackett status = %v, want UP", status.Components["jackett"].Status)
	}
	if status.Components["prowlarr"].Status != StatusUp {
		t.Fatalf("prowlarr status = %v, want UP", status.Components["prowlarr"].Status)
	}
}

func TestSystemStatusMarksFFProbeDownWhenExecutableIsMissing(t *testing.T) {
	service := Service{
		Mode:           "local",
		ActiveProfiles: []string{"local"},
		DBPing:         &mockDBPinger{},
		StoragePath:    t.TempDir(),
		FFProbePath:    "missing-ffprobe",
		QBittorrentURL: "http://127.0.0.1:1",
		SearchProvider: "jackett",
		JackettURL:     "http://127.0.0.1:1",
		execLookPath: func(command string) (string, error) {
			return "", os.ErrNotExist
		},
		ProbeTimeout: 50 * time.Millisecond,
	}

	status := service.Status(context.Background())
	if status.Components["ffprobe"].Status != StatusDown {
		t.Fatalf("ffprobe status = %v, want DOWN", status.Components["ffprobe"].Status)
	}
}
