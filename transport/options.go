package transport

import (
	"github.com/luma/pharos/storage"
	"go.uber.org/zap"
)

type Options struct {
	// Host to listen on
	Host string

	// Port to listen on
	Port int

	// Reuseport controls setting SO_REUSEPORT
	// TODO(rolly) this https://blog.cloudflare.com/graceful-upgrades-in-go/
	// TODO(rolly) Reuseport should default to true
	Reuseport bool

	// Trace will dump packets to stdout. This is only useful in local debugging
	Trace bool

	UseStdlib bool

	NumListeners int

	Store storage.Store

	Log *zap.Logger
}
