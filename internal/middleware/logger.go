package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a structured logging middleware.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		attrs := []slog.Attr{
			slog.String("method", method),
			slog.Int("status", statusCode),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", clientIP),
			slog.String("user_agent", userAgent),
			slog.Duration("latency", latency),
			slog.Int("body_size", bodySize),
		}

		if requestID := GetRequestID(c); requestID != "" {
			attrs = append(attrs, slog.String("request_id", requestID))
		}

		if userID := GetUserID(c); userID != "" {
			attrs = append(attrs, slog.String("user_id", userID))
		}

		// Convert []slog.Attr to []any for slog methods
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}

		switch {
		case statusCode >= 500:
			slog.Error("Server error", args...)
		case statusCode >= 400:
			slog.Warn("Client error", args...)
		default:
			slog.Info("Request completed", args...)
		}
	}
}
