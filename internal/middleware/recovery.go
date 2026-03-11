package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery returns a panic recovery middleware with structured logging.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())

				attrs := []any{
					slog.Any("error", err),
					slog.String("stack", stack),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				}

				if requestID := GetRequestID(c); requestID != "" {
					attrs = append(attrs, slog.String("request_id", requestID))
				}

				slog.Error("Panic recovered", attrs...)

				requestID := GetRequestID(c)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       "INTERNAL_ERROR",
						"message":    "An unexpected error occurred",
						"request_id": requestID,
					},
				})
			}
		}()
		c.Next()
	}
}

// RecoveryWithWriter is a recovery middleware that also logs panic to a custom location.
func RecoveryWithWriter(panicHandler func(c *gin.Context, err any)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())

				slog.Error("Panic recovered",
					slog.Any("error", err),
					slog.String("stack", stack),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				)

				if panicHandler != nil {
					panicHandler(c, err)
				} else {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": gin.H{
							"code":    "INTERNAL_ERROR",
							"message": fmt.Sprintf("Panic: %v", err),
						},
					})
				}
			}
		}()
		c.Next()
	}
}
