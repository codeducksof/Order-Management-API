package mocks

import (
	"context"
	"order-management-api/internal/domain"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of domain.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockOrderRepository is a mock implementation of domain.OrderRepository.
type MockOrderRepository struct {
	mock.Mock
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{}
}

func (m *MockOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Order, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Order), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// MockOrderCache is a mock implementation of domain.OrderCache.
type MockOrderCache struct {
	mock.Mock
}

func NewMockOrderCache() *MockOrderCache {
	return &MockOrderCache{}
}

func (m *MockOrderCache) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderCache) SetOrder(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderCache) DeleteOrder(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockExternalAPIClient is a mock implementation of domain.ExternalAPIClient.
type MockExternalAPIClient struct {
	mock.Mock
}

func NewMockExternalAPIClient() *MockExternalAPIClient {
	return &MockExternalAPIClient{}
}

func (m *MockExternalAPIClient) CreateOrderRef(ctx context.Context, orderID string) (string, error) {
	args := m.Called(ctx, orderID)
	return args.String(0), args.Error(1)
}
