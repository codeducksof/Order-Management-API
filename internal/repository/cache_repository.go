package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"order-management-api/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const orderCachePrefix = "order:"
const orderCacheTTL = 300 // 5 minutes

type cacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(client *redis.Client) (domain.CacheRepository, domain.OrderCache) {
	r := &cacheRepository{client: client}
	return r, r
}

func (r *cacheRepository) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *cacheRepository) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	return r.client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (r *cacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// OrderCacheKey returns cache key for order by ID
func OrderCacheKey(id string) string {
	return orderCachePrefix + id
}

// GetOrder gets order from cache (convenience for service layer)
func (r *cacheRepository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	key := OrderCacheKey(id)
	val, err := r.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *cacheRepository) SetOrder(ctx context.Context, order *domain.Order) error {
	key := OrderCacheKey(order.ID)
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}
	return r.Set(ctx, key, string(data), orderCacheTTL)
}

func (r *cacheRepository) DeleteOrder(ctx context.Context, id string) error {
	return r.Delete(ctx, OrderCacheKey(id))
}
