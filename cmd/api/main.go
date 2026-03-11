package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"order-management-api/internal/config"
	"order-management-api/internal/logger"
	"order-management-api/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (optional, will not fail if not present)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize logger
	env := os.Getenv("GIN_MODE")
	if env == "" {
		env = "development"
	}
	logger.Init(env)

	slog.Info("Starting Order Management API",
		slog.String("port", cfg.Server.Port),
		slog.String("env", env),
	)

	// Initialize server (DB, Redis, routes, etc.)
	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("Failed to initialize server", slog.Any("error", err))
		os.Exit(1)
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server listening", slog.String("addr", ":"+cfg.Server.Port))
		if err := srv.Start(); err != nil {
			slog.Error("Server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.Any("error", err))
	}

	slog.Info("Server exited")
}
