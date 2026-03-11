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

	"order-management-api/internal/config"
	"order-management-api/internal/domain"
	"order-management-api/internal/handler"
	"order-management-api/internal/logger"
	"order-management-api/internal/middleware"
	"order-management-api/internal/repository"
	"order-management-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}

	// Run migrations
	if err := autoMigrate(db); err != nil {
		slog.Error("Failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize Redis
	rdb, err := initRedis(cfg)
	if err != nil {
		slog.Error("Failed to connect to Redis", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize repositories
	cacheRepo, orderCache := repository.NewCacheRepository(rdb)
	_ = cacheRepo // available for generic cache operations

	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	extAPI := repository.NewMockExternalAPIClient(os.Getenv("EXTERNAL_API_URL"))

	// Initialize services
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	orderSvc := service.NewOrderService(orderRepo, orderCache, extAPI)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)
	healthHandler := handler.NewHealthHandler(db, rdb)

	// Setup Gin
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := setupRouter(cfg, authHandler, orderHandler, healthHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.Any("error", err))
	}

	// Close database connection
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	// Close Redis connection
	rdb.Close()

	slog.Info("Server exited")
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}

	if os.Getenv("GIN_MODE") != "release" {
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), gormConfig)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	return db, nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&domain.User{}, &domain.Order{})
}

func setupRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	orderHandler *handler.OrderHandler,
	healthHandler *handler.HealthHandler,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORSDefault())
	router.Use(middleware.RateLimit(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst))

	// Health check endpoints (no auth required)
	router.GET("/health", healthHandler.Health)
	router.GET("/health/detail", healthHandler.HealthDetail)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/live", healthHandler.Live)

	// Auth endpoints (no auth required)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)

	// Protected API endpoints
	api := router.Group("/api")
	api.Use(middleware.AuthRequired(cfg.JWT.Secret))
	{
		api.POST("/orders", orderHandler.CreateOrder)
		api.GET("/orders", orderHandler.ListOrders)
		api.GET("/orders/:id", orderHandler.GetOrder)
		api.PATCH("/orders/:id/status", orderHandler.UpdateOrderStatus)
	}

	return router
}
