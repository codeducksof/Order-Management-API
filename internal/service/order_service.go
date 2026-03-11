package service

import (
	"context"
	"errors"
	"order-management-api/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrUnauthorized  = errors.New("unauthorized: order does not belong to user")
	ErrInvalidStatus = errors.New("invalid order status")
)

type OrderService struct {
	orderRepo   domain.OrderRepository
	orderCache  domain.OrderCache
	externalAPI domain.ExternalAPIClient
}

func NewOrderService(
	orderRepo domain.OrderRepository,
	orderCache domain.OrderCache,
	externalAPI domain.ExternalAPIClient,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		orderCache:  orderCache,
		externalAPI: externalAPI,
	}
}

type CreateOrderInput struct {
	CustomerName string  `json:"customer_name" binding:"required"`
	TotalAmount  float64 `json:"total_amount" binding:"required,gt=0"`
}

type UpdateOrderStatusInput struct {
	Status domain.OrderStatus `json:"status" binding:"required,oneof=pending confirmed shipped delivered cancelled"`
}

type OrderListResponse struct {
	Orders []*domain.Order `json:"orders"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func (s *OrderService) Create(ctx context.Context, userID string, input CreateOrderInput) (*domain.Order, error) {
	order := &domain.Order{
		ID:           uuid.New().String(),
		UserID:       userID,
		CustomerName: input.CustomerName,
		TotalAmount:  input.TotalAmount,
		Status:       domain.OrderStatusPending,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Integration: mock external API to get reference
	ref, err := s.externalAPI.CreateOrderRef(ctx, order.ID)
	if err == nil {
		order.ExternalRef = ref
		// Optionally persist ExternalRef in DB in a real app
	}

	// Cache the new order
	_ = s.orderCache.SetOrder(ctx, order)

	return order, nil
}

func (s *OrderService) GetByID(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	// Try cache first (Redis)
	order, err := s.orderCache.GetOrder(ctx, orderID)
	if err == nil {
		if order.UserID != userID {
			return nil, ErrUnauthorized
		}
		return order, nil
	}
	if err != redis.Nil {
		// Log cache error but continue to DB
	}

	order, err = s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if order.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Populate cache
	_ = s.orderCache.SetOrder(ctx, order)

	return order, nil
}

func (s *OrderService) GetByUserID(ctx context.Context, userID string, limit, offset int) (*OrderListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	orders, total, err := s.orderRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	return &OrderListResponse{
		Orders: orders,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, orderID, userID string, input UpdateOrderStatusInput) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if order.UserID != userID {
		return nil, ErrUnauthorized
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, input.Status); err != nil {
		return nil, err
	}

	order.Status = input.Status

	// Invalidate cache so next read gets fresh data
	_ = s.orderCache.DeleteOrder(ctx, orderID)

	return order, nil
}
