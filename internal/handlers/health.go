package handlers

import (
	"net/http"
	"os/exec"

	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// HealthResponse represents the health status of the application.
type HealthResponse struct {
	Status       string `json:"status"`
	Dependencies struct {
		Ytdlp bool `json:"ytdlp"`
	} `json:"dependencies"`
}

// HealthHandler handles the health check request.
func HealthHandler(w http.ResponseWriter, r *http.Request) error {
	res := HealthResponse{
		Status: "ok",
	}

	_, err := exec.LookPath("yt-dlp")
	res.Dependencies.Ytdlp = (err == nil)

	if !res.Dependencies.Ytdlp {
		res.Status = "degraded"
	}

	httputils.WriteJSON(w, http.StatusOK, res)
	return nil
}
