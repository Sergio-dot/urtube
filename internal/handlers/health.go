package handlers

import (
	"net/http"

	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// HealthResponse represents the health status of the application.
type HealthResponse struct {
	// Status is the general health status of the application (e.g., ok, degraded).
	Status string `json:"status"`
	// Dependencies contains the health status of the application's dependencies.
	Dependencies struct {
		// Ytdlp indicates if the yt-dlp executable is available.
		Ytdlp bool `json:"ytdlp"`
		// Version is the version of the yt-dlp executable.
		Version string `json:"version,omitempty"`
	} `json:"dependencies"`
}

// HealthHandler reports the dependency status determined at startup.
type HealthHandler struct {
	// YtdlpVersion is the yt-dlp version detected at startup. Empty means
	// yt-dlp is unavailable.
	YtdlpVersion string
}

// HandleHealth handles the health check request.
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) error {
	res := HealthResponse{Status: "ok"}
	res.Dependencies.Ytdlp = h.YtdlpVersion != ""
	res.Dependencies.Version = h.YtdlpVersion
	if !res.Dependencies.Ytdlp {
		res.Status = "degraded"
	}

	httputils.WriteJSON(w, http.StatusOK, res)
	return nil
}
