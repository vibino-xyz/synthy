package shttp

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type Server struct {
	httpServer *http.Server
}

func NewServeMux() *http.ServeMux {
	return http.NewServeMux()
}

func NewServer(mux *http.ServeMux) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: mux,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
