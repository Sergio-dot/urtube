package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadyzURL(t *testing.T) {
	tests := []struct {
		host, want string
	}{
		{"", "http://127.0.0.1:8080/readyz"},
		{"0.0.0.0", "http://127.0.0.1:8080/readyz"},
		{"::", "http://127.0.0.1:8080/readyz"},
		{"192.168.1.10", "http://192.168.1.10:8080/readyz"},
		{"::1", "http://[::1]:8080/readyz"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, readyzURL(tt.host, "8080"), "host %q", tt.host)
	}
}

func TestProbe(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()

	assert.NoError(t, probe(context.Background(), srv.URL))

	status = http.StatusServiceUnavailable
	assert.ErrorContains(t, probe(context.Background(), srv.URL), "503")

	srv.Close()
	assert.Error(t, probe(context.Background(), srv.URL), "server down")
}
