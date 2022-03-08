package internal

import (
	"context"
	"net"
	"net/http"
	"time"
)

type listenConfig struct {
	net.ListenConfig
}
type httpServer struct {
	http.Server
}

func (h *httpServer) Serve() error {
	return nil
}

func newHttpServer() *httpServer {
	return &httpServer{
		Server: http.Server{
			ReadHeaderTimeout: 20 * time.Second,
			ReadTimeout:       1 * time.Minute,
			WriteTimeout:      2 * time.Minute,
		},
	}
}

func newListenConfig() *listenConfig {
	return &listenConfig{}
}

func newListener(l *listenConfig) (net.Listener, error) {
	return l.Listen(context.Background(), "tcp", ":8080")
}
