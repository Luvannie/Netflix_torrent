package system

import (
	"net/http"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/httpx"
)

func Handler(service Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := service.Status(r.Context())
		api.WriteOK(w, http.StatusOK, status, httpx.EffectiveRequestID(r))
	})
}