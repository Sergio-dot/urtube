package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/go-chi/chi/v5"
)

// search handles GET /api/v1/search/{query}?limit=&wantLiveStreams=&source=.
// Defaults and caps for limit are the search service's business.
func (a *api) search(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	includeLive, _ := strconv.ParseBool(q.Get("wantLiveStreams"))

	videos, err := a.Search.Search(r.Context(), media.SearchQuery{
		Text:        chi.URLParam(r, "query"),
		Limit:       limit,
		IncludeLive: includeLive,
		Source:      media.SearchSource(q.Get("source")),
	})
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

// suggestTags handles GET /api/v1/tags?url=, the tags urtube would use for
// a download, for the user to check before starting it.
func (a *api) suggestTags(w http.ResponseWriter, r *http.Request) error {
	sug, err := a.Suggest.Suggest(r.Context(), r.URL.Query().Get("url"))
	if err != nil {
		return err
	}
	known := sug.KnownArtists
	if known == nil {
		known = []string{}
	}
	writeJSON(w, http.StatusOK, suggestionResponse{Tags: newTagsDTO(sug.Tags), KnownArtists: known})
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

// getSettings handles GET /api/v1/settings.
func (a *api) getSettings(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, a.settingsResponse())
	return nil
}

// updateSettings handles PUT /api/v1/settings.
func (a *api) updateSettings(w http.ResponseWriter, r *http.Request) error {
	var body settingsRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return statusError{status: http.StatusBadRequest, msg: "invalid request body"}
	}

	if body.DownloadDir != nil {
		if err := a.Settings.SetDownloadDir(strings.TrimSpace(*body.DownloadDir)); err != nil {
			return err
		}
	}
	if j := body.Jellyfin; j != nil {
		if err := a.Settings.SetJellyfin(j.URL, j.APIKey); err != nil {
			return err
		}
	}
	writeJSON(w, http.StatusOK, a.settingsResponse())
	return nil
}

// testJellyfin handles POST /api/v1/settings/jellyfin/test: it checks a URL
// and API key before they are saved. Without apiKey, the saved key is used.
func (a *api) testJellyfin(w http.ResponseWriter, r *http.Request) error {
	var body jellyfinRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return statusError{status: http.StatusBadRequest, msg: "invalid request body"}
	}

	_, key := a.Settings.Jellyfin()
	if body.APIKey != nil {
		key = strings.TrimSpace(*body.APIKey)
	}
	url := strings.TrimSpace(body.URL)
	if url == "" || key == "" {
		return statusError{status: http.StatusBadRequest, msg: "enter the Jellyfin URL and API key first"}
	}

	msg, err := a.CheckJellyfin(r.Context(), url, key)
	if err != nil {
		return statusError{status: http.StatusBadGateway, msg: err.Error()}
	}
	writeJSON(w, http.StatusOK, jellyfinTestResponse{Message: "Connected to " + msg})
	return nil
}

func (a *api) settingsResponse() settingsResponse {
	dir, err := a.Settings.DownloadDir()
	jURL, jKey := a.Settings.Jellyfin()
	res := settingsResponse{
		DownloadDir:        dir,
		DefaultDownloadDir: a.Settings.DefaultDownloadDir(),
		DownloadRoot:       a.Settings.Root(),
		Jellyfin: jellyfinResponse{
			URL:       jURL,
			APIKeySet: jKey != "",
			Enabled:   jURL != "" && jKey != "",
		},
	}
	if err != nil {
		res.DownloadDirError = err.Error()
	}
	return res
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
