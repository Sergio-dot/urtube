package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/server"
	"github.com/stretchr/testify/assert"
)

const (
	addr = "localhost:0"
)

func TestServer(t *testing.T) {
	t.Run("create server", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		s, err := server.NewServer(addr, handler)
		assert.NoError(t, err)
		defer s.Stop(context.Background())

		assert.NotNil(t, s)
	})

	t.Run("start and stop server", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		s, err := server.NewServer(addr, handler)
		assert.NoError(t, err)

		errCh := make(chan error, 1)
		go func() {
			errCh <- s.Start()
		}()

		resp, err := http.Get("http://" + s.Addr())
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		assert.NoError(t, s.Stop(ctx))

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("server did not stop in time")
		}
	})
}
