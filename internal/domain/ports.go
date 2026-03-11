package domain

import "context"

type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}

// OrderCache is used by order service for caching order by ID
type OrderCache interface {
	GetOrder(ctx context.Context, id string) (*Order, error)
	SetOrder(ctx context.Context, order *Order) error
	DeleteOrder(ctx context.Context, id string) error
}

type ExternalAPIClient interface {
	CreateOrderRef(ctx context.Context, orderID string) (string, error)
}
