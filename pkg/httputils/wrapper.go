package httputils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// APIError is an optional custom error to hold HTTP status codes.
type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return e.Message
}

// APIFunc is a custom signature for handlers that allows returning an error.
type APIFunc func(w http.ResponseWriter, r *http.Request) error

// MakeHandler adapts an APIFunc into a standard http.HandlerFunc
func MakeHandler(h APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			var apiErr APIError
			if errors.As(err, &apiErr) {
				Error(w, apiErr.StatusCode, apiErr.Message)
			} else {
				// unexpected internal error
				log.Printf("internal server error: %v", err)
				Error(w, http.StatusInternalServerError, "internal server error")
			}
		}
	}
}

// WriteJSON writes a JSON response with a given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	if data == nil {
		w.WriteHeader(status)
		return
	}

	buf, err := json.Marshal(data)
	if err != nil {
		log.Printf("failed to marshal json: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf)
}

// Error writes an error response with a given status code.
func Error(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
