package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Config struct {
	TargetBaseURL string
	LocalToken    string
}

func NewReverseProxy(cfg Config) http.Handler {
	target, err := url.Parse(cfg.TargetBaseURL)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "invalid upstream target", http.StatusInternalServerError)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director

	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		if needsLocalToken(r.Method, r.URL.Path) && strings.TrimSpace(cfg.LocalToken) != "" {
			r.Header.Set("X-App-Local-Token", cfg.LocalToken)
		}
	}

	return proxy
}

func needsLocalToken(method, path string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	return strings.HasPrefix(path, "/api/v1/search/jobs") ||
		strings.HasPrefix(path, "/api/v1/downloads") ||
		strings.HasPrefix(path, "/api/v1/settings")
}
