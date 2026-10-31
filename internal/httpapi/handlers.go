package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// search handles GET /api/v1/search/{query}?limit=&wantLiveStreams=.
// Defaults and caps for limit are the search service's business.
func (a *api) search(w http.ResponseWriter, r *http.Request) error {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	includeLive, _ := strconv.ParseBool(r.URL.Query().Get("wantLiveStreams"))

	videos, err := a.Search.Search(r.Context(), chi.URLParam(r, "query"), limit, includeLive)
	if err != nil {
		return err
	}

	resp := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		resp = append(resp, newVideoResponse(v))
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

// startDownload handles POST /api/v1/download.
func (a *api) startDownload(w http.ResponseWriter, r *http.Request) error {
	var body downloadRequest
	dec := json.NewDecoder(r.Body)
	// Reject unknown fields so stale clients still sending raw yt-dlp flags
	// fail loudly instead of having them silently ignored.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return statusError{status: http.StatusBadRequest, msg: "invalid request body"}
	}

	d, err := a.Downloads.Start(body.toMedia())
	if err != nil {
		return err
	}

	writeJSON(w, http.StatusAccepted, downloadResponse{UUID: d.ID, Message: "download started"})
	return nil
}

// cancelDownload handles DELETE /api/v1/download/{id}.
func (a *api) cancelDownload(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if err := a.Downloads.Cancel(id); err != nil {
		return err
	}

	writeJSON(w, http.StatusOK, downloadResponse{UUID: id, Message: "download cancelled"})
	return nil
}

type healthResponse struct {
	// Status is ok, or degraded when yt-dlp is unavailable.
	Status       string `json:"status"`
	Dependencies struct {
		Ytdlp   bool   `json:"ytdlp"`
		Version string `json:"version,omitempty"`
	} `json:"dependencies"`
}

// health handles GET /api/v1/health with the dependency status found at
// startup.
func (a *api) health(w http.ResponseWriter, r *http.Request) error {
	res := healthResponse{Status: "ok"}
	res.Dependencies.Ytdlp = a.YtdlpVersion != ""
	res.Dependencies.Version = a.YtdlpVersion
	if !res.Dependencies.Ytdlp {
		res.Status = "degraded"
	}

	writeJSON(w, http.StatusOK, res)
	return nil
}
