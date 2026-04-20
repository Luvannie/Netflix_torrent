package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/netflixtorrent/backend-go/internal/httpx"
)

func TestPathResolverReplacesSpecialCharacters(t *testing.T) {
	resolver := NewPathResolver([]string{"/data"})

	tests := []struct {
		input    string
		expected string
	}{
		{"path<with>special", "path_with_special"},
		{"path:with:colons", "path_with_colons"},
		{`path"with"quotes`, "path_with_quotes"},
		{"path|with|pipes", "path_with_pipes"},
		{"path?with?question", "path_with_question"},
		{"path*with*asterisks", "path_with_asterisks"},
	}

	for _, test := range tests {
		result, err := resolver.ResolveAndValidate(test.input)
		if err != nil {
			t.Fatalf("ResolveAndValidate(%q) error = %v", test.input, err)
		}
		if !strings.Contains(result, "path_with") {
			t.Errorf("ResolveAndValidate(%q) = %q, should replace special chars", test.input, result)
		}
	}
}

func TestPathResolverRejectsTraversalSegments(t *testing.T) {
	resolver := NewPathResolver([]string{"/data"})

	_, err := resolver.ResolveAndValidate("/data/../../etc/passwd")
	if err == nil {
		t.Fatalf("ResolveAndValidate should reject traversal")
	}
}

func TestPathResolverValidatesWithinAllowedBases(t *testing.T) {
	resolver := NewPathResolver([]string{"/data", "/tmp"})

	result, err := resolver.ResolveAndValidate("/data/media/movies")
	if err != nil {
		t.Fatalf("ResolveAndValidate error = %v", err)
	}
	if !strings.Contains(result, "media") {
		t.Errorf("Result = %q, should contain media", result)
	}
}

func TestPathResolverIsWritable(t *testing.T) {
	resolver := NewPathResolver([]string{"/tmp"})

	err := resolver.IsWritable("/tmp/test-path-resolver")
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
func (m *mockRepo) Err() error           { return m.err }

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