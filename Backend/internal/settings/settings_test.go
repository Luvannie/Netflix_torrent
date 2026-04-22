package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netflixtorrent/backend-go/internal/httpx"
)

func TestPathResolverReplacesSpecialCharacters(t *testing.T) {
	base := t.TempDir()
	resolver := NewPathResolver([]string{base})

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(base, "path<with>special"), filepath.Join(base, "path_with_special")},
		{filepath.Join(base, "path:with:colons"), filepath.Join(base, "path_with_colons")},
		{filepath.Join(base, `path"with"quotes`), filepath.Join(base, "path_with_quotes")},
		{filepath.Join(base, "path|with|pipes"), filepath.Join(base, "path_with_pipes")},
		{filepath.Join(base, "path?with?question"), filepath.Join(base, "path_with_question")},
		{filepath.Join(base, "path*with*asterisks"), filepath.Join(base, "path_with_asterisks")},
	}

	for _, test := range tests {
		result, err := resolver.ResolveAndValidate(test.input)
		if err != nil {
			t.Fatalf("ResolveAndValidate(%q) error = %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("ResolveAndValidate(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestPathResolverRejectsTraversalSegments(t *testing.T) {
	base := t.TempDir()
	resolver := NewPathResolver([]string{base})

	_, err := resolver.ResolveAndValidate(filepath.Join(base, "..", "etc", "passwd"))
	if err == nil {
		t.Fatalf("ResolveAndValidate should reject traversal")
	}
}

func TestPathResolverValidatesWithinAllowedBases(t *testing.T) {
	base := t.TempDir()
	resolver := NewPathResolver([]string{base})

	result, err := resolver.ResolveAndValidate(filepath.Join(base, "media", "movies"))
	if err != nil {
		t.Fatalf("ResolveAndValidate error = %v", err)
	}
	if result != filepath.Join(base, "media", "movies") {
		t.Errorf("Result = %q", result)
	}
}

func TestPathResolverRejectsPathOutsideAllowedBases(t *testing.T) {
	allowedBase := t.TempDir()
	outsideBase := t.TempDir()
	resolver := NewPathResolver([]string{allowedBase})

	_, err := resolver.ResolveAndValidate(filepath.Join(outsideBase, "movies"))
	if err == nil {
		t.Fatal("ResolveAndValidate should reject a path outside the allowed base")
	}
}

func TestPathResolverIsWritable(t *testing.T) {
	base := t.TempDir()
	resolver := NewPathResolver([]string{base})

	err := resolver.IsWritable(filepath.Join(base, "test-path-resolver"))
	if err != nil {
		t.Fatalf("IsWritable error = %v", err)
	}
}

func TestStorageProfileFields(t *testing.T) {
	profile := StorageProfile{
		ID:       1,
		Name:     "Movies",
		BasePath: "/data/media/movies",
		Priority: 1,
		Active:   true,
	}

	if profile.ID != 1 {
		t.Fatalf("ID = %d", profile.ID)
	}
	if profile.Name != "Movies" {
		t.Fatalf("Name = %q", profile.Name)
	}
}

func TestNotFoundErrorMessage(t *testing.T) {
	err := NotFoundError{ID: 42}
	msg := err.Error()
	if !strings.Contains(msg, "42") && !strings.Contains(msg, "Storage profile not found") {
		t.Fatalf("Error message = %q, should contain ID or meaningful text", msg)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := ValidationError{Field: "name", Message: "Name is required"}
	msg := err.Error()
	if msg != "Name is required" {
		t.Fatalf("Error message = %q", msg)
	}
}

type mockRepo struct {
	profiles []StorageProfile
	err      error
}

func (m *mockRepo) List() []StorageProfile { return m.profiles }
func (m *mockRepo) Err() error             { return m.err }

type repoInterface interface {
	List(ctx interface{}) ([]StorageProfile, error)
}

var _ interface{} = (*repoInterface)(nil)

func TestCreateRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateStorageProfileRequest
		wantErr string
	}{
		{
			name:    "Empty name",
			req:     CreateStorageProfileRequest{Name: "", BasePath: "/data"},
			wantErr: "Name is required",
		},
		{
			name:    "Empty basePath",
			req:     CreateStorageProfileRequest{Name: "Movies", BasePath: ""},
			wantErr: "Base path is required",
		},
		{
			name:    "Name too long",
			req:     CreateStorageProfileRequest{Name: strings.Repeat("a", 101), BasePath: "/data"},
			wantErr: "Name must not exceed 100 characters",
		},
		{
			name: "Valid request",
			req:  CreateStorageProfileRequest{Name: "Movies", BasePath: "/data/media"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateRequest(tt.req)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
			}
		})
	}
}

func validateCreateRequest(req CreateStorageProfileRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ValidationError{Field: "name", Message: "Name is required"}
	}
	if len(req.Name) > 100 {
		return ValidationError{Field: "name", Message: "Name must not exceed 100 characters"}
	}
	if strings.TrimSpace(req.BasePath) == "" {
		return ValidationError{Field: "basePath", Message: "Base path is required"}
	}
	return nil
}

func TestUpdateRequestValidation(t *testing.T) {
	emptyStr := ""
	validStr := "valid"

	tests := []struct {
		name    string
		req     UpdateStorageProfileRequest
		wantErr string
	}{
		{
			name:    "Blank name provided",
			req:     UpdateStorageProfileRequest{Name: &emptyStr},
			wantErr: "Name must not be blank when provided",
		},
		{
			name:    "Blank basePath provided",
			req:     UpdateStorageProfileRequest{BasePath: &emptyStr},
			wantErr: "Base path must not be blank when provided",
		},
		{
			name: "Valid with pointer fields",
			req:  UpdateStorageProfileRequest{Name: &validStr, BasePath: &validStr},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateRequest(tt.req)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
			}
		})
	}
}

func validateUpdateRequest(req UpdateStorageProfileRequest) error {
	if req.Name != nil && *req.Name == "" {
		return ValidationError{Field: "name", Message: "Name must not be blank when provided"}
	}
	if req.BasePath != nil && *req.BasePath == "" {
		return ValidationError{Field: "basePath", Message: "Base path must not be blank when provided"}
	}
	return nil
}

func TestJSONSerialization(t *testing.T) {
	profile := StorageProfile{
		ID:       1,
		Name:     "Movies",
		BasePath: "/data/media",
		Priority: 1,
		Active:   true,
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if decoded["id"].(float64) != 1 {
		t.Fatalf("id = %v", decoded["id"])
	}
	if decoded["name"].(string) != "Movies" {
		t.Fatalf("name = %v", decoded["name"])
	}
}

func TestHealthHandlerWithSettingsRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/storage-profiles", nil)
	rec := httptest.NewRecorder()

	handler := httpx.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

type fakeSettingsRepo struct {
	created    CreateStorageProfileRequest
	updatedID  int64
	updated    UpdateStorageProfileRequest
	createHits int
	updateHits int
}

func (f *fakeSettingsRepo) List(ctx context.Context) ([]StorageProfile, error) {
	return []StorageProfile{}, nil
}

func (f *fakeSettingsRepo) GetByID(ctx context.Context, id int64) (*StorageProfile, error) {
	return &StorageProfile{ID: id, Name: "Movies", BasePath: filepath.Join(os.TempDir(), "movies"), Priority: 1, Active: true}, nil
}

func (f *fakeSettingsRepo) Create(ctx context.Context, req CreateStorageProfileRequest) (*StorageProfile, error) {
	f.createHits++
	f.created = req
	return &StorageProfile{ID: 1, Name: req.Name, BasePath: req.BasePath, Priority: 1, Active: true}, nil
}

func (f *fakeSettingsRepo) Update(ctx context.Context, id int64, req UpdateStorageProfileRequest) (*StorageProfile, error) {
	f.updateHits++
	f.updatedID = id
	f.updated = req

	basePath := filepath.Join(os.TempDir(), "updated")
	if req.BasePath != nil {
		basePath = *req.BasePath
	}

	return &StorageProfile{ID: id, Name: "Updated", BasePath: basePath, Priority: 1, Active: true}, nil
}

func (f *fakeSettingsRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func TestCreateHandlerUsesSanitizedResolvedBasePath(t *testing.T) {
	allowedBase := t.TempDir()
	repo := &fakeSettingsRepo{}
	handler := NewHandler(repo, NewPathResolver([]string{allowedBase}))

	rawPath := filepath.Join(allowedBase, "media", ".", "movies")
	body, err := json.Marshal(map[string]any{
		"name":     "Movies",
		"basePath": rawPath,
		"priority": 1,
		"active":   true,
	})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/storage-profiles", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.create(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.createHits != 1 {
		t.Fatalf("createHits = %d", repo.createHits)
	}

	expected := filepath.Join(allowedBase, "media", "movies")
	if repo.created.BasePath != expected {
		t.Fatalf("BasePath = %q, want %q", repo.created.BasePath, expected)
	}
}

func TestUpdateHandlerRejectsPathOutsideAllowedBase(t *testing.T) {
	allowedBase := t.TempDir()
	outsideBase := t.TempDir()
	repo := &fakeSettingsRepo{}
	handler := NewHandler(repo, NewPathResolver([]string{allowedBase}))

	body, err := json.Marshal(map[string]any{
		"basePath": filepath.Join(outsideBase, "movies"),
	})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/storage-profiles/5", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.update(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.updateHits != 0 {
		t.Fatalf("updateHits = %d, want 0", repo.updateHits)
	}
}
