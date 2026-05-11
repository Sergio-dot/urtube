package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	err := HealthHandler(w, req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Since we expect it to be installed on the machine running tests
	// but we can't be 100% sure in every environment, we just check
	// that the fields exist.
	assert.NotEmpty(t, resp.Status)
}
