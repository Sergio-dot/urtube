package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		addr      string
		handler   http.HandlerFunc
		expectErr bool
	}{
		{
			name:      "create server",
			addr:      "localhost:0",
			handler:   http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			expectErr: false,
		},
		{
			name:      "empty addr",
			addr:      "",
			handler:   http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			expectErr: true,
		},
		{
			name:      "invalid addr",
			addr:      "[IP_ADDRESS]",
			handler:   http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := server.NewServer(tt.addr, tt.handler)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}

func TestAddr(t *testing.T) {
	s, err := server.NewServer("localhost:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	assert.NoError(t, err)
	assert.NotEqual(t, "localhost:0", s.Addr())
}

func TestStart_GracefulShutdown(t *testing.T) {
	s, err := server.NewServer("localhost:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	assert.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	// Give the server a moment to bind and start accepting
	time.Sleep(10 * time.Millisecond)

	assert.NoError(t, s.Stop(context.Background()))

	// Start() must return nil on graceful shutdown
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return within the deadline")
	}
}
