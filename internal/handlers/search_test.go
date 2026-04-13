package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/go-chi/chi/v5"
	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
)

type mockSearcher struct {
	MockSearch func(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error)
}

func (m *mockSearcher) Search(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
	return m.MockSearch(ctx, param)
}

func TestSearchMediaSuccess(t *testing.T) {
	mock := &mockSearcher{
		MockSearch: func(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
			return []*ytdlp.ExtractedInfo{{Title: strutils.StringPtr("Test Video")}}, nil
		},
	}
	handler := &SearchHandler{Searcher: mock}

	req := httptest.NewRequest(http.MethodGet, "/search/test+query", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("searchParam", "test video")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	httputils.MakeHandler(handler.SearchMedia).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Video")
}

func TestSearchMediaEmpty(t *testing.T) {
	mock := &mockSearcher{
		MockSearch: func(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
			return []*ytdlp.ExtractedInfo{{Title: strutils.StringPtr("Test Video")}}, nil
		},
	}
	handler := &SearchHandler{Searcher: mock}

	req := httptest.NewRequest(http.MethodGet, "/search/", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("searchParam", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	httputils.MakeHandler(handler.SearchMedia).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "search parameter is required")
}

func TestSearchMediaNoResults(t *testing.T) {
	mock := &mockSearcher{
		MockSearch: func(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
			return nil, search.ErrNoResults
		},
	}
	handler := &SearchHandler{Searcher: mock}

	req := httptest.NewRequest(http.MethodGet, "/search/test+query", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("searchParam", "test video")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	httputils.MakeHandler(handler.SearchMedia).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "no results found")
}
