package downloads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type QBittorrentClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client

	mu            sync.Mutex
	sessionCookie string
}

type QBittorrentAPIError struct {
	Message string
	Err     error
}

func (e *QBittorrentAPIError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return e.Message + ": " + e.Err.Error()
}

func (e *QBittorrentAPIError) Unwrap() error {
	return e.Err
}

func NewQBittorrentClient(baseURL string, username string, password string) *QBittorrentClient {
	return NewQBittorrentClientWithHTTPClient(baseURL, username, password, &http.Client{Timeout: 30 * time.Second})
}

func NewQBittorrentClientWithHTTPClient(baseURL string, username string, password string, client *http.Client) *QBittorrentClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &QBittorrentClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		client:   client,
	}
}

func (c *QBittorrentClient) AddTorrent(ctx context.Context, torrentURLOrPath string, savePath string) (string, error) {
	return c.postForm(ctx, "/api/v2/torrents/add", url.Values{
		"urls":     {torrentURLOrPath},
		"savepath": {savePath},
	})
}

func (c *QBittorrentClient) PollStatus(ctx context.Context, hash string) (TorrentStatus, error) {
	cookie, err := c.withSessionCookie(ctx)
	if err != nil {
		return TorrentStatus{}, err
	}

	endpoint := c.baseURL + "/api/v2/torrents/info?hashes=" + url.QueryEscape(hash)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TorrentStatus{}, err
	}
	req.Header.Set("Cookie", cookie)

	resp, err := c.client.Do(req)
	if err != nil {
		return TorrentStatus{}, &QBittorrentAPIError{Message: "failed to poll torrent status", Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TorrentStatus{}, &QBittorrentAPIError{Message: "failed to read torrent status", Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TorrentStatus{}, &QBittorrentAPIError{Message: fmt.Sprintf("qBittorrent status failed: HTTP %d", resp.StatusCode)}
	}

	var infos []torrentInfoResponse
	if err := json.Unmarshal(body, &infos); err != nil {
		return TorrentStatus{}, &QBittorrentAPIError{Message: "failed to decode torrent status", Err: err}
	}
	if len(infos) == 0 {
		return TorrentStatus{}, &QBittorrentAPIError{Message: "torrent not found: " + hash}
	}
	return infos[0].toStatus(), nil
}

func (c *QBittorrentClient) Version(ctx context.Context) (string, error) {
	cookie, err := c.withSessionCookie(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/version", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", cookie)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", &QBittorrentAPIError{Message: "failed to query qBittorrent version", Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &QBittorrentAPIError{Message: fmt.Sprintf("qBittorrent version failed: HTTP %d", resp.StatusCode)}
	}
	return string(body), nil
}

func (c *QBittorrentClient) Pause(ctx context.Context, hash string) error {
	_, err := c.postForm(ctx, "/api/v2/torrents/pause", url.Values{"hashes": {hash}})
	return err
}

func (c *QBittorrentClient) Resume(ctx context.Context, hash string) error {
	_, err := c.postForm(ctx, "/api/v2/torrents/resume", url.Values{"hashes": {hash}})
	return err
}

func (c *QBittorrentClient) Delete(ctx context.Context, hash string) error {
	_, err := c.postForm(ctx, "/api/v2/torrents/delete", url.Values{"hashes": {hash}})
	return err
}

func (c *QBittorrentClient) postForm(ctx context.Context, path string, form url.Values) (string, error) {
	cookie, err := c.withSessionCookie(ctx)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", &QBittorrentAPIError{Message: "qBittorrent request failed", Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &QBittorrentAPIError{Message: fmt.Sprintf("qBittorrent request failed: HTTP %d", resp.StatusCode)}
	}
	return string(body), nil
}

func (c *QBittorrentClient) withSessionCookie(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionCookie != "" {
		return c.sessionCookie, nil
	}

	form := url.Values{
		"username": {c.username},
		"password": {c.password},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", &QBittorrentAPIError{Message: "qBittorrent login failed", Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &QBittorrentAPIError{Message: fmt.Sprintf("qBittorrent login failed: HTTP %d", resp.StatusCode)}
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" && strings.TrimSpace(cookie.Value) != "" {
			c.sessionCookie = "SID=" + cookie.Value
			return c.sessionCookie, nil
		}
	}

	return "", &QBittorrentAPIError{Message: "qBittorrent login did not return a SID cookie"}
}

type torrentInfoResponse struct {
	Hash             string  `json:"hash"`
	Name             string  `json:"name"`
	Progress         float64 `json:"progress"`
	DownloadSpeed    int64   `json:"dlspeed"`
	DownloadSpeedAlt int64   `json:"dl_speed"`
	UploadSpeed      int64   `json:"upspeed"`
	UploadSpeedAlt   int64   `json:"ups_speed"`
	NumSeeds         int     `json:"num_seeds"`
	NumPeers         int     `json:"num_peers"`
	State            string  `json:"state"`
	SavePath         string  `json:"save_path"`
	ContentPath      string  `json:"content_path"`
	TotalSize        int64   `json:"total_size"`
	Size             int64   `json:"size"`
}

func (r torrentInfoResponse) toStatus() TorrentStatus {
	downloadSpeed := r.DownloadSpeed
	if downloadSpeed == 0 {
		downloadSpeed = r.DownloadSpeedAlt
	}
	uploadSpeed := r.UploadSpeed
	if uploadSpeed == 0 {
		uploadSpeed = r.UploadSpeedAlt
	}
	peerCount := r.NumSeeds
	if peerCount == 0 {
		peerCount = r.NumPeers
	}
	totalSize := r.TotalSize
	if totalSize == 0 {
		totalSize = r.Size
	}

	return TorrentStatus{
		Hash:          r.Hash,
		Name:          r.Name,
		Progress:      math.Round(r.Progress*10000) / 100,
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
		PeerCount:     peerCount,
		State:         mapQBittorrentState(r.State),
		SavePath:      r.SavePath,
		ContentPath:   r.ContentPath,
		TotalSize:     totalSize,
	}
}

func mapQBittorrentState(state string) string {
	if strings.HasPrefix(state, "downloading") {
		return "downloading"
	}
	if strings.HasPrefix(state, "seeding") {
		return "seeding"
	}
	if state == "pausedUP" || state == "pausedDL" {
		return "paused"
	}
	if state == "queued" || state == "checking" {
		return "queued"
	}
	if state == "error" {
		return "error"
	}
	if state == "" {
		return "unknown"
	}
	return state
}
