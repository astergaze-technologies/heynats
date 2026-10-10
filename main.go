package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astergaze-solutions/heynats/internal/api"
	"github.com/astergaze-solutions/heynats/internal/config"
	"github.com/astergaze-solutions/heynats/internal/infrastructure"
	"github.com/gin-gonic/gin"
)

//go:embed all:client/dist
var clientDist embed.FS

const shutdownTimeout = 15 * time.Second

func main() {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(2)
	}
	setupLogging(cfg)

	dist, err := fs.Sub(clientDist, "client/dist")
	if err != nil {
		slog.Error("embedded client missing", "err", err)
		os.Exit(1)
	}

	natsConnections := api.NewNatsConnection()
	srv := &http.Server{
		Handler:           newRouter(dist, natsConnections),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		slog.Error("listen failed", "addr", cfg.HTTPAddr, "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("starting server", "addr", ln.Addr().String())
	if err := serve(ctx, srv, ln); err != nil {
		slog.Error("server stopped with error", "err", err)
	}
	natsConnections.Shutdown()
	slog.Info("server stopped")
}

func setupLogging(cfg config.Config) {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, opts)
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))

	if cfg.LogLevel > slog.LevelDebug {
		gin.SetMode(gin.ReleaseMode)
	}
}

func newRouter(dist fs.FS, natsConnections *api.NatsConnectionStore) http.Handler {
	router := infrastructure.NewRouter(dist)
	middleware := api.NewConnectionMiddleware(natsConnections)

	apiGroup := router.Group("/api")
	api.NewHealth(apiGroup).RegisterRoutes()

	natsAPIGroup := apiGroup.Group("/nats")
	api.NewHeyNats(natsAPIGroup, natsConnections, middleware).RegisterRoutes()
	api.NewKVAPI(natsAPIGroup, natsConnections, middleware).RegisterRoutes()
	api.NewStreamAPI(natsAPIGroup, natsConnections, middleware).RegisterRoutes()
	api.NewPublishAPI(natsAPIGroup, natsConnections, middleware).RegisterRoutes()
	api.NewSubscribeAPI(natsAPIGroup, natsConnections, middleware).RegisterRoutes()
	return router
}

// serve runs srv until ctx is done, then shuts down gracefully.
func serve(ctx context.Context, srv *http.Server, ln net.Listener) error {
	// Live streams only end when their request context ends, so cancel all
	// request contexts once shutdown starts.
	reqCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	srv.BaseContext = func(net.Listener) context.Context { return reqCtx }
	srv.RegisterOnShutdown(cancelRequests)

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
