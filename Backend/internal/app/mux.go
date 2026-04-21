package app

import (
	"net/http"
	"strings"
)

type ServeMux struct {
	routes map[string]http.Handler
}

func NewServeMux() *ServeMux {
	return &ServeMux{routes: make(map[string]http.Handler)}
}

func (m *ServeMux) Handle(pattern string, handler http.Handler) {
	m.routes[pattern] = handler
}

func (m *ServeMux) Handler(r *http.Request) http.Handler {
	path := r.URL.Path

	for pattern, handler := range m.routes {
		if method, _, ok := strings.Cut(pattern, " "); ok {
			if r.Method != method {
				continue
			}
			if match(path, pattern[len(method)+1:]) {
				return handler
			}
		}
	}

	return http.NotFoundHandler()
}

func match(path, pattern string) bool {
	if !strings.Contains(pattern, "{") {
		return path == pattern
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(pathParts) != len(patternParts) {
		return false
	}

	for i := range pathParts {
		if strings.HasPrefix(patternParts[i], "{") {
			continue
		}
		if pathParts[i] != patternParts[i] {
			return false
		}
	}

	return true
}

func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := m.Handler(r)
	handler.ServeHTTP(w, r)
}