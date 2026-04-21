package httpx

import (
	"net"
	"net/http"
	"strings"

	"github.com/netflixtorrent/backend-go/internal/api"
)

const LocalTokenHeader = "X-App-Local-Token"

func LocalOnlyMiddleware(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled || isLocalRequest(r) {
				next.ServeHTTP(w, r)
				return
			}
			writeRawError(w, http.StatusForbidden, "", "ACCESS_DENIED", "Access denied. This endpoint is only accessible from localhost.")
		})
	}
}

func LocalTokenMiddleware(enabled bool, expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled || !isWriteMethod(r.Method) || !isProtectedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			if strings.TrimSpace(expectedToken) == "" {
				writeRawError(w, http.StatusServiceUnavailable, "", "TOKEN_NOT_CONFIGURED", "Local token is enabled but not configured.")
				return
			}

			token := strings.TrimSpace(r.Header.Get(LocalTokenHeader))
			if token == "" {
				writeRawError(w, http.StatusForbidden, "", "TOKEN_MISSING", "Local token required for this endpoint.")
				return
			}

			if token != expectedToken {
				writeRawError(w, http.StatusForbidden, "", "TOKEN_INVALID", "Invalid local token.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isProtectedPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/search/jobs") ||
		strings.HasPrefix(path, "/api/v1/downloads") ||
		strings.HasPrefix(path, "/api/v1/settings") ||
		strings.HasPrefix(path, "/api/v1/library/scan")
}

func writeRawError(w http.ResponseWriter, status int, requestID, code, message string) {
	api.WriteError(w, status, code, message, nil, requestID)
}