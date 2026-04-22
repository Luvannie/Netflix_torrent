package system

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type Service struct {
	Mode           string
	ActiveProfiles []string
	DBPing         DatabasePinger
	StoragePath    string
	FFProbePath    string
	QBittorrentURL string
	JackettURL     string
	ProwlarrURL    string
	SearchProvider string
	ProbeTimeout   time.Duration

	httpClient   *http.Client
	pathStat     func(string) (os.FileInfo, error)
	execLookPath func(string) (string, error)
}

func (s Service) Status(ctx context.Context) SystemStatusResponse {
	components := map[string]ComponentStatus{
		"database":    s.checkDatabase(ctx),
		"storage":     s.checkStorage(ctx),
		"ffprobe":     s.checkFFProbe(ctx),
		"qbittorrent": s.checkQBittorrent(ctx),
		"jackett":     s.checkJackett(ctx),
		"prowlarr":    s.checkProwlarr(ctx),
	}

	overall := StatusUp
	for _, comp := range components {
		if comp.Status == StatusDown {
			overall = StatusDown
			break
		}
	}

	return SystemStatusResponse{
		OverallStatus:  overall,
		Mode:           s.Mode,
		ActiveProfiles: s.ActiveProfiles,
		Components:     components,
		CheckedAt:     time.Now().UTC(),
	}
}

func (s Service) checkDatabase(ctx context.Context) ComponentStatus {
	if s.DBPing == nil {
		return ComponentStatus{Status: StatusDown, Message: "Database connection is not configured", Details: map[string]any{}}
	}
	if err := s.DBPing.Ping(ctx); err != nil {
		return ComponentStatus{Status: StatusDown, Message: err.Error(), Details: map[string]any{}}
	}
	return ComponentStatus{Status: StatusUp, Message: "Database connection is valid", Details: map[string]any{"databaseProduct": "PostgreSQL"}}
}

func (s Service) checkStorage(ctx context.Context) ComponentStatus {
	if s.StoragePath == "" {
		return ComponentStatus{Status: StatusDown, Message: "Storage path is not configured", Details: map[string]any{}}
	}

	info, err := s.statPath(s.StoragePath)
	if err != nil {
		return ComponentStatus{Status: StatusDown, Message: "Storage path is not accessible: " + err.Error(), Details: map[string]any{"path": s.StoragePath}}
	}
	if !info.IsDir() {
		return ComponentStatus{Status: StatusDown, Message: "Storage path is not a directory", Details: map[string]any{"path": s.StoragePath}}
	}

	return ComponentStatus{Status: StatusUp, Message: "Storage path is accessible", Details: map[string]any{"path": s.StoragePath}}
}

func (s Service) checkFFProbe(ctx context.Context) ComponentStatus {
	if s.FFProbePath == "" {
		return ComponentStatus{Status: StatusDown, Message: "ffprobe path is not configured", Details: map[string]any{}}
	}

	if pathLooksExplicit(s.FFProbePath) {
		if _, err := s.statPath(s.FFProbePath); err != nil {
			return ComponentStatus{Status: StatusDown, Message: "ffprobe executable is not accessible: " + err.Error(), Details: map[string]any{"command": s.FFProbePath}}
		}
	} else if _, err := s.lookPath(s.FFProbePath); err != nil {
		return ComponentStatus{Status: StatusDown, Message: "ffprobe executable was not found in PATH", Details: map[string]any{"command": s.FFProbePath}}
	}

	return ComponentStatus{Status: StatusUp, Message: "ffprobe executable is reachable", Details: map[string]any{"command": s.FFProbePath}}
}

func (s Service) checkQBittorrent(ctx context.Context) ComponentStatus {
	if s.QBittorrentURL == "" {
		return ComponentStatus{Status: StatusDown, Message: "qBittorrent URL is not configured", Details: map[string]any{}}
	}

	return s.probeHTTP(ctx, "qBittorrent", s.QBittorrentURL)
}

func (s Service) checkJackett(ctx context.Context) ComponentStatus {
	if s.SearchProvider == "jackett" || s.SearchProvider == "both" {
		if s.JackettURL == "" {
			return ComponentStatus{Status: StatusDown, Message: "Jackett URL is not configured", Details: map[string]any{"provider": s.SearchProvider}}
		}
		return s.probeHTTP(ctx, "Jackett", s.JackettURL)
	}
	return ComponentStatus{Status: StatusDisabled, Message: "Jackett is not the active provider", Details: map[string]any{"provider": s.SearchProvider}}
}

func (s Service) checkProwlarr(ctx context.Context) ComponentStatus {
	if s.SearchProvider == "prowlarr" || s.SearchProvider == "both" {
		if s.ProwlarrURL == "" {
			return ComponentStatus{Status: StatusDown, Message: "Prowlarr URL is not configured", Details: map[string]any{"provider": s.SearchProvider}}
		}
		return s.probeHTTP(ctx, "Prowlarr", s.ProwlarrURL)
	}
	return ComponentStatus{Status: StatusDisabled, Message: "Prowlarr is not the active provider", Details: map[string]any{"provider": s.SearchProvider}}
}

func (s Service) statPath(path string) (os.FileInfo, error) {
	if s.pathStat != nil {
		return s.pathStat(path)
	}
	return os.Stat(path)
}

func (s Service) lookPath(command string) (string, error) {
	if s.execLookPath != nil {
		return s.execLookPath(command)
	}
	return exec.LookPath(command)
}

func (s Service) probeHTTP(ctx context.Context, component string, rawURL string) ComponentStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(rawURL, "/"), nil)
	if err != nil {
		return ComponentStatus{Status: StatusDown, Message: fmt.Sprintf("%s URL is invalid: %v", component, err), Details: map[string]any{"url": rawURL}}
	}

	resp, err := s.httpProbeClient().Do(req)
	if err != nil {
		return ComponentStatus{Status: StatusDown, Message: fmt.Sprintf("%s is unreachable: %v", component, err), Details: map[string]any{"url": rawURL}}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		return ComponentStatus{Status: StatusDown, Message: fmt.Sprintf("%s responded with HTTP %d", component, resp.StatusCode), Details: map[string]any{"url": rawURL, "httpStatus": resp.StatusCode}}
	}

	return ComponentStatus{Status: StatusUp, Message: fmt.Sprintf("%s is reachable", component), Details: map[string]any{"url": rawURL, "httpStatus": resp.StatusCode}}
}

func (s Service) httpProbeClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}

	timeout := s.ProbeTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	return &http.Client{Timeout: timeout}
}

func pathLooksExplicit(path string) bool {
	return strings.ContainsRune(path, filepath.Separator) || filepath.VolumeName(path) != ""
}
