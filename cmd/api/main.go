package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DanielCahya/url-shortener/internal/auth"
	"github.com/DanielCahya/url-shortener/internal/config"
	"github.com/DanielCahya/url-shortener/internal/handler"
	"github.com/DanielCahya/url-shortener/internal/logger"
	"github.com/DanielCahya/url-shortener/internal/middleware"
	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1. Configuration loading
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Logger initialization
	log := logger.New("url-shortener-api", cfg.LogLevel)
	log.Info("initializing application",
		slog.String("env", cfg.AppEnv),
		slog.String("port", cfg.ServerPort),
	)

	// 3. Database initialization
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := repository.NewPostgresPool(ctx, cfg)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()
	log.Info("connected to PostgreSQL successfully")

	// 4. Run database migrations
	if err := repository.RunMigrations(ctx, dbPool); err != nil {
		log.Error("failed to run database migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("database migrations verified and applied")

	// 5. Repository construction
	urlRepo := repository.NewPostgresURLRepository(dbPool)
	authRepo := repository.NewPostgresAuthRepository(dbPool)

	// 6. Service construction
	tokenService := auth.NewTokenService(auth.JWTConfig{
		SecretKey:  cfg.JWTSecret,
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		Issuer:     "url-shortener",
	})
	urlService := url.NewService(urlRepo, cfg.BaseURL)
	authService := auth.NewAuthService(authRepo, tokenService) // no tokenService needed for register

	// 7. Handler construction
	urlHandler := url.NewHandler(urlService)
	authHandler := auth.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(dbPool)

	// 8. Router and Middleware construction
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(log))
	r.Use(chiMiddleware.Recoverer)

	// Register Routes
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)
	r.Post("/api/v1/urls", urlHandler.Create)
	r.Post("/api/v1/auth/register", authHandler.RegisterUser)
	r.Post("/api/v1/auth/login", authHandler.LoginUser)
	r.Get("/{short_code}", urlHandler.Redirect)

	// 9. HTTP server startup
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP server starting", slog.String("addr", server.Addr))
		serverErrors <- server.ListenAndServe()
	}()

	// 10. Graceful shutdown listening to OS signals
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server encountered fatal error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-shutdownSignal:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server forced to shutdown", slog.String("error", err.Error()))
			_ = server.Close()
		}
		log.Info("server gracefully stopped")
	}
}
