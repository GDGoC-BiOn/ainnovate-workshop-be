package platform

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorDetail is the body of every error response.
//
// Example:
//
//	{
//	  "error": {
//	    "code": "INVALID_REQUEST",
//	    "message": "lesson_id is required"
//	  }
//	}
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorBody wraps the error detail so responses have a stable shape.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// WriteJSON writes a JSON response with the given HTTP status code.
// It is the single place where JSON responses are encoded.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The response is already written; there is nothing sensible left to do.
		// Encoding failures are extremely rare (e.g. broken connection).
		http.Error(w, "internal encoding error", http.StatusInternalServerError)
	}
}

// WriteError writes a JSON error response with a stable structure.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// DecodeJSON reads the request body into dst, rejecting unknown JSON fields
// so a typo in the API contract is caught early instead of silently ignored.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB limit

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	return nil
}
