package handlers

import (
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
}

func TestMakeHandler(t *testing.T) {
	t.Run("wrap handler", func(t *testing.T) {
		h := MakeHandler(func(w http.ResponseWriter, r *http.Request) error {
			w.WriteHeader(http.StatusOK)
			return nil
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestJSON(t *testing.T) {
	t.Run("write json", func(t *testing.T) {
		w := httptest.NewRecorder()
		JSON(w, http.StatusOK, map[string]string{"message": "ok"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
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
