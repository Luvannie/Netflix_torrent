package health

import (
	"net/http"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/httpx"
)

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteOK(w, http.StatusOK, map[string]string{
			"status":  "UP",
			"service": "backend",
		}, httpx.InboundRequestID(r))
	})
}