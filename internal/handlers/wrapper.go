package handlers

import (
	"encoding/json"
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
			if apiErr, ok := err.(APIError); ok {
				Error(w, apiErr.StatusCode, apiErr.Message)
			} else {
				// unexpected internal error
				log.Printf("internal server error: %v", err)
				Error(w, http.StatusInternalServerError, "internal server error")
			}
		}
	}
}

// JSON writes a JSON response with a given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
