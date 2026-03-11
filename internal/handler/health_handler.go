package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

// HealthStatus represents the health status of a component.
type HealthStatus struct {
	Status    string `json:"status"`
	Latency   string `json:"latency,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HealthResponse represents the overall health response.
type HealthResponse struct {
	Status     string                  `json:"status"`
	Version    string                  `json:"version"`
	Uptime     string                  `json:"uptime"`
	GoVersion  string                  `json:"go_version"`
	Components map[string]HealthStatus `json:"components"`
}

var startTime = time.Now()

// Health returns a basic health check.
// @Summary      Health check
// @Description  Returns basic health status
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// HealthDetail returns a detailed health check.
// @Summary      Detailed health check
// @Description  Returns detailed health status including database and Redis connectivity
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health/detail [get]
func (h *HealthHandler) HealthDetail(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:     "healthy",
		Version:    "1.0.0",
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		GoVersion:  runtime.Version(),
		Components: make(map[string]HealthStatus),
	}

	// Check database
	dbStatus := h.checkDatabase(ctx)
	response.Components["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check Redis
	redisStatus := h.checkRedis(ctx)
	response.Components["redis"] = redisStatus
	if redisStatus.Status != "healthy" {
		response.Status = "degraded"
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// Ready checks if the service is ready to accept traffic.
// @Summary      Readiness check
// @Description  Returns whether the service is ready to accept traffic
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Check database connectivity
	if err := h.pingDatabase(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "database not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Live checks if the service is alive.
// @Summary      Liveness check
// @Description  Returns whether the service is alive
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

func (h *HealthHandler) checkDatabase(ctx context.Context) HealthStatus {
	start := time.Now()
	err := h.pingDatabase(ctx)
	latency := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

func (h *HealthHandler) pingDatabase(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (h *HealthHandler) checkRedis(ctx context.Context) HealthStatus {
	start := time.Now()
	err := h.redis.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Latency: latency.String(),
	}
}
