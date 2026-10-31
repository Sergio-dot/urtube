package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedStatus string
		expectedYtdlp  bool
	}{
		{name: "reports the detected version", version: "2026.10.01", expectedStatus: "ok", expectedYtdlp: true},
		{name: "degraded without yt-dlp", version: "", expectedStatus: "degraded", expectedYtdlp: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HealthHandler{YtdlpVersion: tt.version}
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			assert.NoError(t, h.HandleHealth(w, req))
			assert.Equal(t, http.StatusOK, w.Code)

			var resp HealthResponse
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tt.expectedStatus, resp.Status)
			assert.Equal(t, tt.expectedYtdlp, resp.Dependencies.Ytdlp)
			assert.Equal(t, tt.version, resp.Dependencies.Version)
		})
	}
}
