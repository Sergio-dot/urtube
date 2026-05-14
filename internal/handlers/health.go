package handlers

import (
	"bytes"
	"net/http"
	"os/exec"
	"strings"

	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// HealthResponse represents the health status of the application.
type HealthResponse struct {
	Status       string `json:"status"`
	Dependencies struct {
		Ytdlp   bool   `json:"ytdlp"`
		Version string `json:"version,omitempty"`
	} `json:"dependencies"`
}

// HealthHandler handles the health check request.
func HealthHandler(w http.ResponseWriter, r *http.Request) error {
	res := HealthResponse{
		Status: "ok",
	}

	path, err := exec.LookPath("yt-dlp")
	res.Dependencies.Ytdlp = (err == nil)

	if !res.Dependencies.Ytdlp {
		res.Status = "degraded"
	} else {
		cmd := exec.Command(path, "--version")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			res.Dependencies.Version = strings.TrimSpace(out.String())
		}
	}

	httputils.WriteJSON(w, http.StatusOK, res)
	return nil
}
