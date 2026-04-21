package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netflixtorrent/backend-go/internal/app"
	"github.com/netflixtorrent/backend-go/internal/catalog"
	"github.com/netflixtorrent/backend-go/internal/config"
	"github.com/netflixtorrent/backend-go/internal/database"
	"github.com/netflixtorrent/backend-go/internal/downloads"
	"github.com/netflixtorrent/backend-go/internal/events"
	"github.com/netflixtorrent/backend-go/internal/health"
	"github.com/netflixtorrent/backend-go/internal/httpx"
	"github.com/netflixtorrent/backend-go/internal/library"
	"github.com/netflixtorrent/backend-go/internal/search"
	"github.com/netflixtorrent/backend-go/internal/settings"
	"github.com/netflixtorrent/backend-go/internal/streaming"
	"github.com/netflixtorrent/backend-go/internal/system"
	"github.com/netflixtorrent/backend-go/internal/websocket"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx := context.Background()

	postgresURL, err := cfg.Database.PostgresURL()
	if err != nil {
		logger.Error("failed to build postgres URL", "error", err)
		os.Exit(1)
	}

	pool, err := database.Open(ctx, postgresURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	if err := database.ApplyMigrations(ctx, pool); err != nil {
		logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	hub := websocket.NewHub()
	eventBus := events.NewBus()

	mux := app.NewServeMux()

	mux.Handle("GET /api/v1/health", health.Handler())

	searchRepo := search.NewRepository(pool)
	searchService := search.NewService(
		searchRepo,
		nil,
		nil,
		cfg.Search.Provider,
	)
	searchHandler := search.NewHandler(searchService)
	for pattern, handler := range searchHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	downloadsRepo := downloads.NewRepository(pool)
	downloadsService := downloads.NewService(
		downloadsRepo,
		nil,
		nil,
	)
	downloadsHandler := downloads.NewHandler(downloadsService)
	for pattern, handler := range downloadsHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	catalogService := catalog.NewService(
		catalog.NewRepository(pool),
		catalog.NewTMDBClient(
			cfg.TMDB.Key,
			cfg.TMDB.ReadAccessToken,
			cfg.TMDB.BaseURL,
			cfg.TMDB.ImageBaseURL,
		),
		cfg.TMDB.ImageBaseURL,
	)
	catalogHandler := catalog.NewHandler(catalogService)
	for pattern, handler := range catalogHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	settingsRepo := settings.NewRepository(pool)
	settingsHandler := settings.NewHandler(settingsRepo, settings.NewPathResolver([]string{}))
	for pattern, handler := range settingsHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	systemService := system.Service{
		Mode:           cfg.Mode,
		ActiveProfiles: cfg.ActiveProfiles,
		DBPing:         pool,
		StoragePath:    cfg.Download.DefaultSavePath,
		FFProbePath:    cfg.FFprobe.Path,
		QBittorrentURL: cfg.QBittorrent.BaseURL,
		SearchProvider: cfg.Search.Provider,
	}
	systemHandler := system.Handler(systemService)
	mux.Handle("GET /api/v1/system/status", systemHandler)

	libraryRepo := library.NewRepository(pool)
	libraryHandler := library.NewHandler(libraryRepo)
	for pattern, handler := range libraryHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	streamingService := streaming.NewService(libraryRepo, cfg.FFprobe.Path, cfg.Download.DefaultSavePath)
	streamingHandler := streaming.NewHandler(streamingService)
	for pattern, handler := range streamingHandler.Routes() {
		mux.Handle(pattern, handler)
	}

	mux.Handle("GET /ws", websocket.ServeWS(hub, eventBus))

	var finalHandler http.Handler = mux

	if cfg.Auth.LocalTokenEnabled {
		finalHandler = httpx.LocalTokenMiddleware(true, cfg.Auth.LocalToken)(finalHandler)
	}

	finalHandler = httpx.RequestIDMiddleware(finalHandler)

	addr := app.Address(cfg.Server.Address, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}