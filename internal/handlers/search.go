package handlers

import (
	"errors"
	"net/http"

	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/go-chi/chi/v5"
)

// SearchHandler is the handler for the search video request
type SearchHandler struct {
	Searcher search.Searcher
}

// SearchVideo handles the search video request
func (h *SearchHandler) SearchVideo(w http.ResponseWriter, r *http.Request) error {
	param := chi.URLParam(r, "searchParam")
	if strutils.IsEmpty(param) {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: "search parameter is required"}
	}

	videos, err := h.Searcher.Search(r.Context(), param)
	if err != nil {
		if errors.Is(err, search.ErrNoResults) {
			return httputils.APIError{StatusCode: http.StatusNotFound, Message: err.Error()}
		}
		return err
	}

	httputils.WriteJSON(w, http.StatusOK, videos)
	return nil
}
