package httpx

import (
	"net/http"
	"strings"
)

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && originAllowed(origin, allowedOrigins) && strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		trimmed := strings.TrimSpace(item)
		if trimmed == "*" || trimmed == origin {
			return true
		}
	}
	return false
}