package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJellyfin accepts only key and counts library refreshes.
func fakeJellyfin(t *testing.T, key string, refreshes *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != `MediaBrowser Token="`+key+`"` {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Home","Version":"10.11.2","Id":"x"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/Library/Refresh":
			refreshes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClient(t *testing.T) {
	var refreshes atomic.Int32
	srv := fakeJellyfin(t, "k3y", &refreshes)
	ctx := context.Background()

	t.Run("check describes the server", func(t *testing.T) {
		got, err := Client{URL: srv.URL + "/", APIKey: "k3y"}.Check(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Home (Jellyfin 10.11.2)", got)
	})

	t.Run("refresh", func(t *testing.T) {
		require.NoError(t, Client{URL: srv.URL, APIKey: "k3y"}.Refresh(ctx))
		assert.Equal(t, int32(1), refreshes.Load())
	})

	t.Run("wrong key", func(t *testing.T) {
		_, err := Client{URL: srv.URL, APIKey: "nope"}.Check(ctx)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("not Jellyfin", func(t *testing.T) {
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html>router login</html>"))
		}))
		defer other.Close()
		_, err := Client{URL: other.URL, APIKey: "k"}.Check(ctx)
		assert.ErrorContains(t, err, "doesn't look like a Jellyfin server")
	})

	t.Run("unreachable", func(t *testing.T) {
		_, err := Client{URL: "http://127.0.0.1:1", APIKey: "k"}.Check(ctx)
		assert.ErrorContains(t, err, "can't reach Jellyfin")
	})

	t.Run("server error", func(t *testing.T) {
		broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer broken.Close()
		assert.EqualError(t, Client{URL: broken.URL, APIKey: "k"}.Refresh(ctx), "Jellyfin answered 500 Internal Server Error")
	})
}

func TestNotifier(t *testing.T) {
	var refreshes atomic.Int32
	srv := fakeJellyfin(t, "k3y", &refreshes)

	t.Run("does nothing until configured", func(t *testing.T) {
		for _, cfg := range [][2]string{{"", ""}, {srv.URL, ""}, {"", "k3y"}} {
			n := Notifier{Config: func() (string, string) { return cfg[0], cfg[1] }}
			assert.NoError(t, n.LibraryChanged(context.Background()))
		}
		assert.Equal(t, int32(0), refreshes.Load())
	})

	t.Run("refreshes with the current settings", func(t *testing.T) {
		n := Notifier{Config: func() (string, string) { return srv.URL, "k3y" }}
		require.NoError(t, n.LibraryChanged(context.Background()))
		assert.Equal(t, int32(1), refreshes.Load())
	})
}
