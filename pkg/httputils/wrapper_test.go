package httputils

import (
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError(t *testing.T) {
	apiErr := APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "bad request",
	}
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "bad request", apiErr.Message)
	assert.Equal(t, "bad request", apiErr.Error())
	assert.Implements(t, (*error)(nil), apiErr)
}

func TestMakeHandler(t *testing.T) {
	tests := []struct {
		name           string
		handler        APIFunc
		expectedStatus int
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				return APIError{
					StatusCode: http.StatusBadRequest,
					Message:    "bad request",
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				return errors.New("internal error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := MakeHandler(tt.handler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			h.ServeHTTP(w, r)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           any
		expectedStatus int
	}{
		{
			name:           "success",
			status:         http.StatusOK,
			data:           map[string]string{"message": "ok"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "nil data",
			status:         http.StatusOK,
			data:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "marshal error",
			status:         http.StatusInternalServerError,
			data:           math.Inf(1),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.status, tt.data)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{name: "bad request", status: http.StatusBadRequest, message: "bad request"},
		{name: "unauthorized", status: http.StatusUnauthorized, message: "unauthorized"},
		{name: "not found", status: http.StatusNotFound, message: "not found"},
		{name: "internal server error", status: http.StatusInternalServerError, message: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, tt.status, tt.message)
			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		})
	}
}
