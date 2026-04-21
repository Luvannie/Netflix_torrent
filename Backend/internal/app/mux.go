package app

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const pathParamsKey contextKey = "pathParams"

type pathParams map[string]string

func withPathParams(ctx context.Context, params pathParams) context.Context {
	return context.WithValue(ctx, pathParamsKey, params)
}

func PathValue(r *http.Request, key string) string {
	params, _ := r.Context().Value(pathParamsKey).(pathParams)
	if params == nil {
		return ""
	}
	return params[key]
}

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
			if params, ok := match(path, pattern[len(method)+1:]); ok {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					r = r.WithContext(withPathParams(r.Context(), params))
					handler.ServeHTTP(w, r)
				})
			}
		}
	}

	return http.NotFoundHandler()
}

func match(path, pattern string) (pathParams, bool) {
	if !strings.Contains(pattern, "{") {
		if path == pattern {
			return nil, true
		}
		return nil, false
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(pathParts) != len(patternParts) {
		return nil, false
	}

	params := make(pathParams)
	for i := range pathParts {
		if strings.HasPrefix(patternParts[i], "{") {
			key := strings.Trim(patternParts[i], "{}")
			params[key] = pathParts[i]
			continue
		}
		if pathParts[i] != patternParts[i] {
			return nil, false
		}
	}

	return params, true
}

func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := m.Handler(r)
	handler.ServeHTTP(w, r)
}
