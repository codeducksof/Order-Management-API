package service

import (
	"context"
	"errors"
	"order-management-api/internal/domain"
	"order-management-api/internal/mocks"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestOrderService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		userID        string
		input         CreateOrderInput
		setupMock     func(*mocks.MockOrderRepository, *mocks.MockOrderCache, *mocks.MockExternalAPIClient)
		expectedError error
		checkResult   func(*testing.T, *domain.Order)
	}{
		{
			name:   "successful order creation",
			userID: "user-123",
			input: CreateOrderInput{
				CustomerName: "John Doe",
				TotalAmount:  99.99,
			},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, extAPI *mocks.MockExternalAPIClient) {
				repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)
				extAPI.On("CreateOrderRef", ctx, mock.AnythingOfType("string")).Return("EXT-123", nil)
				cache.On("SetOrder", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, order *domain.Order) {
				assert.NotNil(t, order)
				assert.NotEmpty(t, order.ID)
				assert.Equal(t, "user-123", order.UserID)
				assert.Equal(t, "John Doe", order.CustomerName)
				assert.Equal(t, 99.99, order.TotalAmount)
				assert.Equal(t, domain.OrderStatusPending, order.Status)
				assert.Equal(t, "EXT-123", order.ExternalRef)
			},
		},
		{
			name:   "repository error",
			userID: "user-123",
			input: CreateOrderInput{
				CustomerName: "John Doe",
				TotalAmount:  99.99,
			},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, extAPI *mocks.MockExternalAPIClient) {
				repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(errors.New("database error"))
			},
			expectedError: errors.New("database error"),
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache, mockExtAPI)

			svc := NewOrderService(mockRepo, mockCache, mockExtAPI)
			order, err := svc.Create(ctx, tt.userID, tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, order)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		orderID       string
		userID        string
		setupMock     func(*mocks.MockOrderRepository, *mocks.MockOrderCache)
		expectedError error
		checkResult   func(*testing.T, *domain.Order)
	}{
		{
			name:    "successful get from cache",
			orderID: "order-123",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", ctx, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123",
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, order *domain.Order) {
				assert.NotNil(t, order)
				assert.Equal(t, "order-123", order.ID)
				assert.Equal(t, "user-123", order.UserID)
			},
		},
		{
			name:    "cache miss, get from database",
			orderID: "order-123",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", ctx, "order-123").Return(nil, redis.Nil)
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123",
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
				cache.On("SetOrder", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, order *domain.Order) {
				assert.NotNil(t, order)
				assert.Equal(t, "order-123", order.ID)
			},
		},
		{
			name:    "order not found",
			orderID: "nonexistent",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", ctx, "nonexistent").Return(nil, redis.Nil)
				repo.On("GetByID", ctx, "nonexistent").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrOrderNotFound,
			checkResult:   nil,
		},
		{
			name:    "unauthorized - different user",
			orderID: "order-123",
			userID:  "different-user",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", ctx, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123", // Different from requesting user
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
			},
			expectedError: ErrUnauthorized,
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache)

			svc := NewOrderService(mockRepo, mockCache, mockExtAPI)
			order, err := svc.GetByID(ctx, tt.orderID, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, order)
			}

			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		userID        string
		limit         int
		offset        int
		setupMock     func(*mocks.MockOrderRepository)
		expectedError error
		checkResult   func(*testing.T, *OrderListResponse)
	}{
		{
			name:   "successful list orders",
			userID: "user-123",
			limit:  10,
			offset: 0,
			setupMock: func(repo *mocks.MockOrderRepository) {
				repo.On("GetByUserID", ctx, "user-123", 10, 0).Return([]*domain.Order{
					{ID: "order-1", UserID: "user-123", CustomerName: "Customer 1", TotalAmount: 10.00},
					{ID: "order-2", UserID: "user-123", CustomerName: "Customer 2", TotalAmount: 20.00},
				}, int64(2), nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, resp *OrderListResponse) {
				assert.NotNil(t, resp)
				assert.Len(t, resp.Orders, 2)
				assert.Equal(t, int64(2), resp.Total)
				assert.Equal(t, 10, resp.Limit)
				assert.Equal(t, 0, resp.Offset)
			},
		},
		{
			name:   "default limit when zero",
			userID: "user-123",
			limit:  0,
			offset: 0,
			setupMock: func(repo *mocks.MockOrderRepository) {
				repo.On("GetByUserID", ctx, "user-123", 20, 0).Return([]*domain.Order{}, int64(0), nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, resp *OrderListResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, 20, resp.Limit)
			},
		},
		{
			name:   "repository error",
			userID: "user-123",
			limit:  10,
			offset: 0,
			setupMock: func(repo *mocks.MockOrderRepository) {
				repo.On("GetByUserID", ctx, "user-123", 10, 0).Return(nil, int64(0), errors.New("database error"))
			},
			expectedError: errors.New("database error"),
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo)

			svc := NewOrderService(mockRepo, mockCache, mockExtAPI)
			resp, err := svc.GetByUserID(ctx, tt.userID, tt.limit, tt.offset)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, resp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		orderID       string
		userID        string
		setupMock     func(*mocks.MockOrderRepository, *mocks.MockOrderCache)
		expectedError error
	}{
		{
			name:    "successful delete pending order",
			orderID: "order-123",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:     "order-123",
					UserID: "user-123",
					Status: domain.OrderStatusPending,
				}, nil)
				repo.On("Delete", ctx, "order-123").Return(nil)
				cache.On("DeleteOrder", ctx, "order-123").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:    "successful delete cancelled order",
			orderID: "order-456",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-456").Return(&domain.Order{
					ID:     "order-456",
					UserID: "user-123",
					Status: domain.OrderStatusCancelled,
				}, nil)
				repo.On("Delete", ctx, "order-456").Return(nil)
				cache.On("DeleteOrder", ctx, "order-456").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:    "cannot delete confirmed order",
			orderID: "order-123",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:     "order-123",
					UserID: "user-123",
					Status: domain.OrderStatusConfirmed,
				}, nil)
			},
			expectedError: ErrCannotDeleteOrder,
		},
		{
			name:    "order not found",
			orderID: "nonexistent",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "nonexistent").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrOrderNotFound,
		},
		{
			name:    "unauthorized - different user",
			orderID: "order-123",
			userID:  "other-user",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:     "order-123",
					UserID: "user-123",
					Status: domain.OrderStatusPending,
				}, nil)
			},
			expectedError: ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache)

			svc := NewOrderService(mockRepo, mockCache, mockExtAPI)
			err := svc.Delete(ctx, tt.orderID, tt.userID)

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestOrderService_UpdateStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		orderID       string
		userID        string
		input         UpdateOrderStatusInput
		setupMock     func(*mocks.MockOrderRepository, *mocks.MockOrderCache)
		expectedError error
		checkResult   func(*testing.T, *domain.Order)
	}{
		{
			name:    "successful status update",
			orderID: "order-123",
			userID:  "user-123",
			input:   UpdateOrderStatusInput{Status: domain.OrderStatusConfirmed},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123",
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
				repo.On("UpdateStatus", ctx, "order-123", domain.OrderStatusConfirmed).Return(nil)
				cache.On("DeleteOrder", ctx, "order-123").Return(nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, order *domain.Order) {
				assert.NotNil(t, order)
				assert.Equal(t, domain.OrderStatusConfirmed, order.Status)
			},
		},
		{
			name:    "order not found",
			orderID: "nonexistent",
			userID:  "user-123",
			input:   UpdateOrderStatusInput{Status: domain.OrderStatusConfirmed},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "nonexistent").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrOrderNotFound,
			checkResult:   nil,
		},
		{
			name:    "unauthorized - different user",
			orderID: "order-123",
			userID:  "different-user",
			input:   UpdateOrderStatusInput{Status: domain.OrderStatusConfirmed},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				repo.On("GetByID", ctx, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123",
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
			},
			expectedError: ErrUnauthorized,
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache)

			svc := NewOrderService(mockRepo, mockCache, mockExtAPI)
			order, err := svc.UpdateStatus(ctx, tt.orderID, tt.userID, tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, order)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
