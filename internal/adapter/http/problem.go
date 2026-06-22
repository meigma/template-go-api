package http

import (
	"encoding/json"
	"net/http"
)

// problemContentType is the RFC 9457 media type for problem-details responses.
const problemContentType = "application/problem+json"

// problemDetails is the RFC 9457 problem document used for error responses that
// originate outside Huma (router fallbacks, panics, timeouts). It mirrors the
// title/status/detail members Huma emits so clients see one consistent error shape.
type problemDetails struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// writeProblem writes an RFC 9457 application/problem+json error response.
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemDetails{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}
