package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/go-chi/chi/v5"
)

// SearchHandler is the handler for the search video request.
type SearchHandler struct {
	Searcher search.Searcher
}

// SearchMedia handles the search media request.
func (h *SearchHandler) SearchMedia(w http.ResponseWriter, r *http.Request) error {
	if h.Searcher == nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "searcher not available"}
	}

	param := chi.URLParam(r, "searchParam")
	if strutils.IsEmpty(param) {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: "search parameter is required"}
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	videos, err := h.Searcher.Search(r.Context(), param, limit)
	if err != nil {
		if errors.Is(err, search.ErrNoResults) {
			return httputils.APIError{StatusCode: http.StatusNotFound, Message: err.Error()}
		}
		return err
	}

	httputils.WriteJSON(w, http.StatusOK, videos)
	return nil
}
