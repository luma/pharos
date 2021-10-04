package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/luma/pharos/internal/env"
	"github.com/luma/pharos/storage"
	"github.com/luma/pharos/transport"
)

var (
	// The host to listen on
	host string

	// The port to listen for http requests on
	httpPort string

	// The port to listen for tcp clients on
	port int
)

func init() {
	flags := StartCmd.PersistentFlags()

	flags.IntVarP(&port, "port", "p", 7363, "The port to listen client connections on")
	flags.StringVar(&httpPort, "http-port", "7362", "The port to listen to HTTP requests on")
	flags.StringVarP(&host, "host", "a", "0.0.0.0", "The host to listen on")
}

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start up the Pharos API service",
	Long: `Start up the Pharos API service

Usage
	pharos start

`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx, signalStop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer signalStop()

		log, err := env.MakeLogger()
		if err != nil {
			return err
		}

		fileLimit, err := setFileLimit()
		if err != nil {
			return err
		}

		log.Info("Set file limit", zap.Uint64("fileLimit", fileLimit))

		conf, err := env.LoadConfig(ctx)
		if err != nil {
			return err
		}

		router := setupRouter(conf.DebugHTTP, log)

		// Ping test
		router.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})

		s := &http.Server{
			Addr:    net.JoinHostPort(host, httpPort),
			Handler: router,
		}

		// Initializing the server in a goroutine so that
		// it won't block the graceful shutdown handling below
		go func() {
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("Http server errored", zap.Error(err))
			}
		}()

		tcp := transport.NewTCP(transport.Options{
			Host:      host,
			Port:      port,
			Reuseport: true,
			Store:     storage.NewInmemoryStore(),
			Log:       log.Named("transport"),
		})

		if err := tcp.Start(ctx); err != nil {
			return err
		}

		log.Info("Listening",
			zap.Any("config", conf),
			zap.String("host", host),
			zap.Int("port", port),
			zap.String("httpPort", httpPort))

		// Listen for the interrupt signal.
		<-ctx.Done()

		// Restore default behavior on the interrupt signal and notify user of shutdown.
		signalStop()
		log.Info("Shutting down gracefully, press Ctrl+C again to force")

		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.SetKeepAlivesEnabled(false)

		if err := s.Shutdown(ctx); err != nil {
			log.Error("Http server forced to shutdown", zap.Error(err))
		}

		if err := tcp.Close(); err != nil {
			log.Error("TCP server forced to shutdown", zap.Error(err))
		}

		log.Info("Exiting")
		return nil
	},
}

func setupRouter(debugHTTP bool, log *zap.Logger) *gin.Engine {
	gin.DisableConsoleColor()
	if !debugHTTP {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Add a ginzap middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	r.Use(ginzap.Ginzap(log, time.RFC3339, true))

	r.Use(ginzap.GinzapWithConfig(log, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/health"},
	}))

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	r.Use(ginzap.RecoveryWithZap(log, true))

	return r
}

func setFileLimit() (uint64, error) {
	var rLimit syscall.Rlimit

	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return 0, err
	}

	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return 0, err
	}

	return rLimit.Cur, nil
}
