package system

import (
	"context"
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
	SearchProvider string
}

func (s Service) Status(ctx context.Context) SystemStatusResponse {
	components := map[string]ComponentStatus{
		"database":   s.checkDatabase(ctx),
		"storage":    s.checkStorage(ctx),
		"ffprobe":    s.checkFFProbe(ctx),
		"qbittorrent": s.checkQBittorrent(ctx),
		"jackett":    s.checkJackett(),
		"prowlarr":   s.checkProwlarr(),
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
	return ComponentStatus{Status: StatusUp, Message: "Storage path is configurable", Details: map[string]any{"path": s.StoragePath}}
}

func (s Service) checkFFProbe(ctx context.Context) ComponentStatus {
	if s.FFProbePath == "" {
		return ComponentStatus{Status: StatusDown, Message: "ffprobe path is not configured", Details: map[string]any{}}
	}
	return ComponentStatus{Status: StatusUp, Message: "ffprobe is configured", Details: map[string]any{"command": s.FFProbePath}}
}

func (s Service) checkQBittorrent(ctx context.Context) ComponentStatus {
	if s.QBittorrentURL == "" {
		return ComponentStatus{Status: StatusDown, Message: "qBittorrent URL is not configured", Details: map[string]any{}}
	}
	return ComponentStatus{Status: StatusUp, Message: "qBittorrent is configured", Details: map[string]any{"url": s.QBittorrentURL}}
}

func (s Service) checkJackett() ComponentStatus {
	if s.SearchProvider == "jackett" || s.SearchProvider == "both" {
		return ComponentStatus{Status: StatusUp, Message: "Jackett is the active torrent search provider", Details: map[string]any{"provider": s.SearchProvider}}
	}
	return ComponentStatus{Status: StatusDisabled, Message: "Jackett is not the active provider", Details: map[string]any{"provider": s.SearchProvider}}
}

func (s Service) checkProwlarr() ComponentStatus {
	if s.SearchProvider == "prowlarr" || s.SearchProvider == "both" {
		return ComponentStatus{Status: StatusUp, Message: "Prowlarr is the active torrent search provider", Details: map[string]any{"provider": s.SearchProvider}}
	}
	return ComponentStatus{Status: StatusDisabled, Message: "Prowlarr is not the active provider", Details: map[string]any{"provider": s.SearchProvider}}
}