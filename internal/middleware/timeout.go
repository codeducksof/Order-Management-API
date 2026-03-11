package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout returns a middleware that sets a timeout on the request context.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{}, 1)

		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			// Request completed normally
		case <-ctx.Done():
			// Timeout exceeded
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": gin.H{
					"code":    "TIMEOUT",
					"message": "Request timeout exceeded",
				},
			})
		}
	}
}
