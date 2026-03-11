package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"order-management-api/internal/config"
	"order-management-api/internal/domain"
	"order-management-api/internal/handler"
	"order-management-api/internal/repository"
	"order-management-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Server holds all dependencies and the HTTP server.
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	db         *gorm.DB
	rdb        *redis.Client
}

// New initializes all dependencies and returns a ready-to-start Server.
func New(cfg *config.Config) (*Server, error) {
	db, err := initDatabase(cfg)
	if err != nil {
		return nil, err
	}

	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	rdb, err := initRedis(cfg)
	if err != nil {
		return nil, err
	}

	// Repositories
	_, orderCache := repository.NewCacheRepository(rdb)
	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	extAPI := repository.NewMockExternalAPIClient(os.Getenv("EXTERNAL_API_URL"))

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	orderSvc := service.NewOrderService(orderRepo, orderCache, extAPI)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)
	healthHandler := handler.NewHealthHandler(db, rdb)

	if os.Getenv("GIN_MODE") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := setupRouter(cfg, authHandler, orderHandler, healthHandler)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &Server{
		cfg:        cfg,
		httpServer: httpServer,
		db:         db,
		rdb:        rdb,
	}, nil
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server and closes all connections.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	if sqlDB, err := s.db.DB(); err == nil {
		sqlDB.Close()
	}

	s.rdb.Close()

	return nil
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
