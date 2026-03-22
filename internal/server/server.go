package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
)

// Server is the server definition
type Server struct {
	server   *http.Server
	listener net.Listener
}

// NewServer creates and binds a TCP listener on addr, returning a Server ready to be started.
func NewServer(addr string, handler http.Handler) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		listener: ln,
	}, nil
}

// Addr returns the listener's network address.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Start begins serving HTTP requests on the listener. It blocks until the server is stopped.
func (s *Server) Start() error {
	log.Print("Server started on port: ", s.server.Addr)
	err := s.server.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop gracefully shuts down the server, waiting for active connections to finish
// before returning. The provided context sets a deadline for the shutdown to complete.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
