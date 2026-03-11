package repository

import (
	"context"
	"fmt"
	"order-management-api/internal/domain"
	"time"
)

// MockExternalAPIClient simulates an external payment/inventory API
type MockExternalAPIClient struct {
	baseURL string
}

func NewMockExternalAPIClient(baseURL string) domain.ExternalAPIClient {
	return &MockExternalAPIClient{baseURL: baseURL}
}

func (c *MockExternalAPIClient) CreateOrderRef(ctx context.Context, orderID string) (string, error) {
	// Simulate external API call - in production would HTTP GET/POST to external service
	ref := fmt.Sprintf("EXT-%s-%d", orderID[:8], time.Now().Unix())
	return ref, nil
}
