package downloads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDownloadTaskStatusTransitions(t *testing.T) {
	validPath := []DownloadTaskStatus{
		StatusRequested,
		StatusSearching,
		StatusSearchReady,
		StatusQueued,
		StatusDownloading,
		StatusPostProcessing,
		StatusStreamReady,
		StatusCompleted,
	}

	for i := 0; i < len(validPath)-1; i++ {
		if !validPath[i].CanTransitionTo(validPath[i+1]) {
			t.Fatalf("%s should transition to %s", validPath[i], validPath[i+1])
		}
	}

	if StatusRequested.CanTransitionTo(StatusQueued) {
		t.Fatalf("REQUESTED should not skip to QUEUED")
	}
	for _, terminal := range []DownloadTaskStatus{StatusCompleted, StatusFailed, StatusCancelled} {
		if !terminal.IsTerminal() {
			t.Fatalf("%s should be terminal", terminal)
		}
		if terminal.CanTransitionTo(StatusRequested) || terminal.CanTransitionTo(StatusFailed) {
			t.Fatalf("%s should not transition to another state", terminal)
		}
	}
	if !StatusDownloading.CanTransitionTo(StatusFailed) || !StatusDownloading.CanTransitionTo(StatusCancelled) {
		t.Fatalf("non-terminal states should allow FAILED and CANCELLED")
	}
}

func TestCreateTaskDefaultsAndAuditTransition(t *testing.T) {
	repo := newMemoryTaskRepository()
	service := NewService(repo, nil, nil)

	task, err := service.CreateTask(context.Background(), 42)
	if err != nil {
		t.Fatalf("CreateTask error = %v", err)
	}

	if task.SearchResultID != 42 {
		t.Fatalf("SearchResultID = %d", task.SearchResultID)
	}
	if task.TorrentHash != "" {
		t.Fatalf("TorrentHash = %q", task.TorrentHash)
	}
	if task.Status != StatusRequested {
		t.Fatalf("Status = %s", task.Status)
	}
	if task.Progress != 0 || task.Speed != 0 || task.PeerCount != 0 {
		t.Fatalf("defaults progress/speed/peers = %v/%d/%d", task.Progress, task.Speed, task.PeerCount)
	}

	transitions := repo.transitions[task.ID]
	if len(transitions) != 1 {
		t.Fatalf("transition count = %d", len(transitions))
	}
	if transitions[0].FromStatus != nil || transitions[0].ToStatus != StatusRequested {
		t.Fatalf("transition = %#v", transitions[0])
	}
}

func TestCancelTaskDeletesTorrentWhenHashPresent(t *testing.T) {
	repo := newMemoryTaskRepository()
	service := NewService(repo, &recordingTorrentClient{}, nil)

	task, err := service.CreateTask(context.Background(), 7)
	if err != nil {
		t.Fatalf("CreateTask error = %v", err)
	}
	task.TorrentHash = "ABC123"
	task.Status = StatusDownloading
	repo.tasks[task.ID] = *task

	client := &recordingTorrentClient{}
	service = NewService(repo, client, nil)
	cancelled, err := service.CancelTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("CancelTask error = %v", err)
	}

	if cancelled.Status != StatusCancelled {
		t.Fatalf("Status = %s", cancelled.Status)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "ABC123" {
		t.Fatalf("deleted hashes = %#v", client.deleted)
	}
}

func TestCreateDownloadHandlerRejectsMissingSearchResultID(t *testing.T) {
	handler := NewHandler(NewService(newMemoryTaskRepository(), nil, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/downloads", strings.NewReader(`{}`))

	handler.Routes()["POST /api/v1/downloads"].ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q", body.Error.Code)
	}
}

func TestListDownloadHandlerReturnsSpringPageShape(t *testing.T) {
	repo := newMemoryTaskRepository()
	service := NewService(repo, nil, nil)
	for i := 0; i < 3; i++ {
		if _, err := service.CreateTask(context.Background(), int64(i+1)); err != nil {
			t.Fatalf("CreateTask error = %v", err)
		}
	}

	handler := NewHandler(service)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?page=0&size=2", nil)

	handler.Routes()["GET /api/v1/downloads"].ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Content          []DownloadTask `json:"content"`
			Number           int            `json:"number"`
			Size             int            `json:"size"`
			TotalElements    int64          `json:"totalElements"`
			TotalPages       int            `json:"totalPages"`
			NumberOfElements int            `json:"numberOfElements"`
			Empty            bool           `json:"empty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Content) != 2 || body.Data.TotalElements != 3 || body.Data.TotalPages != 2 {
		t.Fatalf("page = %#v", body.Data)
	}
}

func TestQBittorrentLoginAndCookieReuse(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	var cookies []string
	var loginForm url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		cookies = append(cookies, r.Header.Get("Cookie"))
		mu.Unlock()

		switch r.URL.Path {
		case "/api/v2/auth/login":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			loginForm = r.PostForm
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session-1"})
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			if r.Header.Get("Cookie") != "SID=session-1" {
				t.Fatalf("add cookie = %q", r.Header.Get("Cookie"))
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			if r.PostForm.Get("urls") != "magnet:?xt=urn:btih:ABC" || r.PostForm.Get("savepath") != "/media" {
				t.Fatalf("add form = %#v", r.PostForm)
			}
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/delete":
			if r.Header.Get("Cookie") != "SID=session-1" {
				t.Fatalf("delete cookie = %q", r.Header.Get("Cookie"))
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewQBittorrentClient(server.URL, "admin", "secret")
	if _, err := client.AddTorrent(context.Background(), "magnet:?xt=urn:btih:ABC", "/media"); err != nil {
		t.Fatalf("AddTorrent error = %v", err)
	}
	if err := client.Delete(context.Background(), "ABC"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}

	if loginForm.Get("username") != "admin" || loginForm.Get("password") != "secret" {
		t.Fatalf("login form = %#v", loginForm)
	}
	if got := strings.Join(paths, ","); got != "/api/v2/auth/login,/api/v2/torrents/add,/api/v2/torrents/delete" {
		t.Fatalf("paths = %s", got)
	}
	if cookies[1] != "SID=session-1" || cookies[2] != "SID=session-1" {
		t.Fatalf("cookies = %#v", cookies)
	}
}

func TestQBittorrentPollStatusMapsProgressAndState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "sid"})
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			if r.URL.Query().Get("hashes") != "HASH1" {
				t.Fatalf("hashes query = %q", r.URL.Query().Get("hashes"))
			}
			_, _ = w.Write([]byte(`[{"hash":"HASH1","name":"Movie","progress":0.42,"dlspeed":123,"upspeed":4,"num_peers":9,"state":"downloading","save_path":"/save","content_path":"/save/Movie.mkv","total_size":1000}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	status, err := NewQBittorrentClient(server.URL, "u", "p").PollStatus(context.Background(), "HASH1")
	if err != nil {
		t.Fatalf("PollStatus error = %v", err)
	}

	if status.Progress != 42 || status.DownloadSpeed != 123 || status.UploadSpeed != 4 || status.PeerCount != 9 {
		t.Fatalf("status metrics = %#v", status)
	}
	if status.State != "downloading" || status.SavePath != "/save" || status.ContentPath != "/save/Movie.mkv" || status.TotalSize != 1000 {
		t.Fatalf("status = %#v", status)
	}
}

func TestQBittorrentPollStatusMissingTorrentReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "sid"})
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := NewQBittorrentClient(server.URL, "u", "p").PollStatus(context.Background(), "missing")
	var apiErr *QBittorrentAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T %v, want QBittorrentAPIError", err, err)
	}
}

type memoryTaskRepository struct {
	tasks       map[int64]DownloadTask
	transitions map[int64][]DownloadStateTransition
	nextID      int64
}

func newMemoryTaskRepository() *memoryTaskRepository {
	return &memoryTaskRepository{
		tasks:       make(map[int64]DownloadTask),
		transitions: make(map[int64][]DownloadStateTransition),
		nextID:      1,
	}
}

func (r *memoryTaskRepository) CreateTask(ctx context.Context, searchResultID int64) (*DownloadTask, error) {
	now := time.Now().UTC()
	task := DownloadTask{
		ID:             r.nextID,
		SearchResultID: searchResultID,
		TorrentHash:    "",
		Status:         StatusRequested,
		Progress:       0,
		Speed:          0,
		PeerCount:      0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.nextID++
	r.tasks[task.ID] = task
	return &task, nil
}

func (r *memoryTaskRepository) GetTask(ctx context.Context, id int64) (*DownloadTask, error) {
	task, ok := r.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return &task, nil
}

func (r *memoryTaskRepository) SaveTask(ctx context.Context, task DownloadTask) (*DownloadTask, error) {
	task.UpdatedAt = time.Now().UTC()
	r.tasks[task.ID] = task
	return &task, nil
}

func (r *memoryTaskRepository) ListTasks(ctx context.Context, limit, offset int) ([]DownloadTask, int64, error) {
	all := make([]DownloadTask, 0, len(r.tasks))
	for id := int64(1); id < r.nextID; id++ {
		if task, ok := r.tasks[id]; ok {
			all = append([]DownloadTask{task}, all...)
		}
	}
	total := int64(len(all))
	if offset >= len(all) {
		return []DownloadTask{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *memoryTaskRepository) RecordTransition(ctx context.Context, taskID int64, from *DownloadTaskStatus, to DownloadTaskStatus, reason string) error {
	r.transitions[taskID] = append(r.transitions[taskID], DownloadStateTransition{
		DownloadTaskID: taskID,
		FromStatus:     from,
		ToStatus:       to,
		Reason:         reason,
		Timestamp:      time.Now().UTC(),
	})
	return nil
}

type recordingTorrentClient struct {
	deleted []string
}

func (c *recordingTorrentClient) Delete(ctx context.Context, hash string) error {
	c.deleted = append(c.deleted, hash)
	return nil
}
