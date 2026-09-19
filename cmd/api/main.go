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
	"github.com/DanielCahya/url-shortener/internal/cache"
	"github.com/DanielCahya/url-shortener/internal/config"
	"github.com/DanielCahya/url-shortener/internal/consumer"
	"github.com/DanielCahya/url-shortener/internal/handler"
	"github.com/DanielCahya/url-shortener/internal/logger"
	"github.com/DanielCahya/url-shortener/internal/middleware"
	"github.com/DanielCahya/url-shortener/internal/queue"
	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/DanielCahya/url-shortener/internal/url"
	"github.com/DanielCahya/url-shortener/internal/worker"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// 5. Initialize Redis Client
	redisClient, err := cache.NewRedisClient(cfg.RedisAddr)
	if err != nil {
		log.Error("failed to connect to redis", slog.String("error", err.Error()))
		// Optionally we could exit, but maybe we want the app to start even if redis is down.
		// However, for this demo we'll exit on failure to ensure environment is fully healthy.
		os.Exit(1)
	}
	defer func() { _ = redisClient.Close() }()
	log.Info("connected to Redis successfully")

	// 5.5 Initialize RabbitMQ Client
	rabbitMQClient, err := queue.NewRabbitMQClient(cfg.RabbitMQURL)
	if err != nil {
		log.Error("failed to connect to rabbitmq", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rabbitMQClient.Close()
	log.Info("connected to RabbitMQ successfully")

	// 6. Repository construction
	urlRepo := repository.NewPostgresURLRepository(dbPool)
	authRepo := repository.NewPostgresAuthRepository(dbPool)
	idempotencyRepo := repository.NewPostgresIdempotencyRepository(dbPool)
	outboxRepo := repository.NewPostgresOutboxRepository(dbPool)
	clickEventRepo := repository.NewPostgresClickEventRepository(dbPool)

	// 6. Service construction
	tokenService := auth.NewTokenService(auth.JWTConfig{
		SecretKey:  cfg.JWTSecret,
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		Issuer:     "url-shortener",
	})

	urlCache := cache.NewRedisURLCache(redisClient)
	analyticsRepo := repository.NewPostgresAnalyticsRepository(outboxRepo)
	urlService := url.NewService(urlRepo, urlCache, analyticsRepo, clickEventRepo, cfg.BaseURL)
	authService := auth.NewAuthService(authRepo, tokenService) // no tokenService needed for register

	// 7. Handler construction
	urlHandler := url.NewHandler(urlService)
	authHandler := auth.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)

	// 8. Worker construction
	cleanupWorker := worker.NewCleanupWorker(idempotencyRepo, 1*time.Hour, log)
	go cleanupWorker.Start()

	outboxPublisher := worker.NewOutboxPublisher(outboxRepo, rabbitMQClient)
	publisherCtx, publisherCancel := context.WithCancel(context.Background())
	go outboxPublisher.Start(publisherCtx)

	analyticsConsumer := consumer.NewAnalyticsConsumer(rabbitMQClient, clickEventRepo)
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go analyticsConsumer.Start(consumerCtx)

	// 9. Router and Middleware construction
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(log))
	r.Use(middleware.Metrics)
	r.Use(chiMiddleware.Recoverer)

	rateLimiter := middleware.RateLimiter(redisClient, cfg)
	idempotencyMiddleware := middleware.Idempotency(redisClient, idempotencyRepo)

	// Register Routes
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	// Frontend routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./frontend/static"))))

	r.With(auth.RequireAuth(tokenService), rateLimiter).Get("/api/v1/urls", urlHandler.List)
	r.With(auth.RequireAuth(tokenService), rateLimiter, idempotencyMiddleware).Post("/api/v1/urls", urlHandler.Create)
	r.With(rateLimiter).Post("/api/v1/auth/register", authHandler.RegisterUser)
	r.With(rateLimiter).Post("/api/v1/auth/login", authHandler.LoginUser)
	r.With(rateLimiter).Post("/api/v1/auth/refresh", authHandler.RefreshTokens)
	r.With(rateLimiter).Post("/api/v1/auth/logout", authHandler.LogoutUser)
	r.With(auth.RequireAuth(tokenService), rateLimiter).Get("/api/v1/auth/me", authHandler.GetProfile)
	r.With(auth.RequireAuth(tokenService), rateLimiter).Delete("/api/v1/urls/{short_code}", urlHandler.Delete)
	r.With(auth.RequireAuth(tokenService), rateLimiter).Get("/api/v1/urls/{short_code}/analytics", urlHandler.GetAnalytics)
	r.With(auth.RequireAuth(tokenService), rateLimiter).Put("/api/v1/urls/{short_code}/enable", urlHandler.UpdateEnabled)
	r.With(rateLimiter).Post("/api/v1/urls/{short_code}/unlock", urlHandler.Unlock)
	r.With(rateLimiter).Get("/{short_code}", urlHandler.Redirect)

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

		cleanupWorker.Stop()
		publisherCancel()
		consumerCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server forced to shutdown", slog.String("error", err.Error()))
			_ = server.Close()
		}
		log.Info("server gracefully stopped")
	}
}
