package server

import (
	"order-management-api/internal/config"
	"order-management-api/internal/handler"
	"order-management-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

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
