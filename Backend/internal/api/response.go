package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type Meta struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID *string   `json:"requestId"`
}

type Response struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
	Meta  Meta     `json:"meta"`
}

type APIError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewMeta(requestID string) Meta {
	var requestIDPointer *string
	if requestID != "" {
		requestIDPointer = &requestID
	}
	return Meta{
		Timestamp: time.Now().UTC(),
		RequestID: requestIDPointer,
	}
}

func WriteOK(w http.ResponseWriter, status int, data any, requestID string) {
	writeJSON(w, status, Response{
		Data: data,
		Meta: NewMeta(requestID),
	})
}

func WriteError(w http.ResponseWriter, status int, code string, message string, details []ErrorDetail, requestID string) {
	if details == nil {
		details = []ErrorDetail{}
	}
	writeJSON(w, status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: NewMeta(requestID),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

func WriteErrorWithStatus(w http.ResponseWriter, status int, requestID string, code string, message string, details []ErrorDetail) {
	if details == nil {
		details = []ErrorDetail{}
	}
	var requestIDPointer *string
	if requestID != "" {
		requestIDPointer = &requestID
	}
	writeJSON(w, status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: Meta{
			Timestamp: time.Now().UTC(),
			RequestID: requestIDPointer,
		},
	})
}