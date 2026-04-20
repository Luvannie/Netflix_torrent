package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const RequestIDHeader = "X-Request-Id"

type contextKey string

const (
	inboundRequestIDKey   contextKey = "inboundRequestID"
	effectiveRequestIDKey contextKey = "effectiveRequestID"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inbound := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		effective := inbound
		if effective == "" {
			effective = newRequestID()
		}

		ctx := context.WithValue(r.Context(), inboundRequestIDKey, inbound)
		ctx = context.WithValue(ctx, effectiveRequestIDKey, effective)
		w.Header().Set(RequestIDHeader, effective)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestID(r *http.Request) string {
	value, _ := r.Context().Value(effectiveRequestIDKey).(string)
	return value
}

func InboundRequestID(r *http.Request) string {
	value, _ := r.Context().Value(inboundRequestIDKey).(string)
	return value
}

func EffectiveRequestID(r *http.Request) string {
	value, _ := r.Context().Value(effectiveRequestIDKey).(string)
	return value
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(bytes[:])
}