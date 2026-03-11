package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"
	// RequestIDCtxKey is the context key for request ID.
	RequestIDCtxKey = "request_id"
)

// RequestID generates a unique request ID for each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDCtxKey, requestID)
		c.Header(RequestIDHeader, requestID)

		// Add to request context for downstream use
		ctx := context.WithValue(c.Request.Context(), RequestIDCtxKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDCtxKey); exists {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
