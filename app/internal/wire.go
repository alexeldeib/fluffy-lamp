//go:build wireinject

package internal

import (
	"github.com/google/wire"
)

func Server() *httpServer {
	wire.Build(newHttpServer)
	return &httpServer{}
}
