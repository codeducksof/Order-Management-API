package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"order-management-api/internal/domain"
	"order-management-api/internal/middleware"
	"order-management-api/internal/mocks"
	"order-management-api/internal/service"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testJWTSecret = "test-secret-key-for-testing-purposes-32-chars"

func generateTestToken(userID, email string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testJWTSecret))
	return tokenString
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		setupMock      func(*mocks.MockUserRepository)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful registration",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"name":     "Test User",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, gorm.ErrRecordNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["token"])
				assert.NotNil(t, resp["user"])
			},
		},
		{
			name: "user already exists",
			body: map[string]interface{}{
				"email":    "existing@example.com",
				"password": "password123",
				"name":     "Existing User",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "existing@example.com").Return(&domain.User{
					ID:    "existing-id",
					Email: "existing@example.com",
				}, nil)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "exists")
			},
		},
		{
			name: "invalid email format",
			body: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password123",
				"name":     "Test User",
			},
			setupMock:      func(m *mocks.MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "password too short",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "12345",
				"name":     "Test User",
			},
			setupMock:      func(m *mocks.MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository()
			tt.setupMock(mockRepo)

			authSvc := service.NewAuthService(mockRepo, testJWTSecret, 24*time.Hour)
			handler := NewAuthHandler(authSvc)

			router := gin.New()
			router.POST("/auth/register", handler.Register)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	tests := []struct {
		name           string
		body           map[string]interface{}
		setupMock      func(*mocks.MockUserRepository)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful login",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "correctpassword",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "test@example.com").Return(&domain.User{
					ID:       "user-id",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["token"])
				assert.NotNil(t, resp["user"])
			},
		},
		{
			name: "invalid credentials",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "wrongpassword",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "test@example.com").Return(&domain.User{
					ID:       "user-id",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
				}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "user not found",
			body: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "nonexistent@example.com").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository()
			tt.setupMock(mockRepo)

			authSvc := service.NewAuthService(mockRepo, testJWTSecret, 24*time.Hour)
			handler := NewAuthHandler(authSvc)

			router := gin.New()
			router.POST("/auth/login", handler.Login)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
		})
	}
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           map[string]interface{}
		setupMock      func(*mocks.MockOrderRepository, *mocks.MockOrderCache, *mocks.MockExternalAPIClient)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "successful order creation",
			userID: "user-123",
			body: map[string]interface{}{
				"customer_name": "John Doe",
				"total_amount":  99.99,
			},
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ext *mocks.MockExternalAPIClient) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)
				ext.On("CreateOrderRef", mock.Anything, mock.AnythingOfType("string")).Return("EXT-123", nil)
				cache.On("SetOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["id"])
				assert.Equal(t, "John Doe", resp["customer_name"])
				assert.Equal(t, 99.99, resp["total_amount"])
				assert.Equal(t, "pending", resp["status"])
			},
		},
		{
			name:   "invalid total amount",
			userID: "user-123",
			body: map[string]interface{}{
				"customer_name": "John Doe",
				"total_amount":  -10.00,
			},
			setupMock:      func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ext *mocks.MockExternalAPIClient) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name:   "missing customer name",
			userID: "user-123",
			body: map[string]interface{}{
				"total_amount": 99.99,
			},
			setupMock:      func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ext *mocks.MockExternalAPIClient) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache, mockExtAPI)

			orderSvc := service.NewOrderService(mockRepo, mockCache, mockExtAPI)
			handler := NewOrderHandler(orderSvc)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.UserIDKey), tt.userID)
				c.Next()
			})
			router.POST("/api/orders", handler.CreateOrder)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
		})
	}
}

func TestOrderHandler_GetOrder(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		userID         string
		setupMock      func(*mocks.MockOrderRepository, *mocks.MockOrderCache)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:    "successful get order",
			orderID: "order-123",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", mock.Anything, "order-123").Return(&domain.Order{
					ID:           "order-123",
					UserID:       "user-123",
					CustomerName: "John Doe",
					TotalAmount:  99.99,
					Status:       domain.OrderStatusPending,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "order-123", resp["id"])
				assert.Equal(t, "John Doe", resp["customer_name"])
			},
		},
		{
			name:    "order not found",
			orderID: "nonexistent",
			userID:  "user-123",
			setupMock: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache) {
				cache.On("GetOrder", mock.Anything, "nonexistent").Return(nil, gorm.ErrRecordNotFound)
				repo.On("GetByID", mock.Anything, "nonexistent").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo, mockCache)

			orderSvc := service.NewOrderService(mockRepo, mockCache, mockExtAPI)
			handler := NewOrderHandler(orderSvc)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.UserIDKey), tt.userID)
				ctx := context.WithValue(c.Request.Context(), middleware.UserIDKey, tt.userID)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			router.GET("/api/orders/:id", handler.GetOrder)

			req := httptest.NewRequest(http.MethodGet, "/api/orders/"+tt.orderID, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
		})
	}
}

func TestOrderHandler_ListOrders(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    string
		setupMock      func(*mocks.MockOrderRepository)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "successful list orders",
			userID:      "user-123",
			queryParams: "?limit=10&offset=0",
			setupMock: func(repo *mocks.MockOrderRepository) {
				repo.On("GetByUserID", mock.Anything, "user-123", 10, 0).Return([]*domain.Order{
					{ID: "order-1", UserID: "user-123", CustomerName: "Customer 1", TotalAmount: 10.00},
					{ID: "order-2", UserID: "user-123", CustomerName: "Customer 2", TotalAmount: 20.00},
				}, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				orders := resp["orders"].([]interface{})
				assert.Len(t, orders, 2)
				assert.Equal(t, float64(2), resp["total"])
			},
		},
		{
			name:        "default pagination",
			userID:      "user-123",
			queryParams: "",
			setupMock: func(repo *mocks.MockOrderRepository) {
				repo.On("GetByUserID", mock.Anything, "user-123", 20, 0).Return([]*domain.Order{}, int64(0), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(20), resp["limit"])
				assert.Equal(t, float64(0), resp["offset"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockOrderRepository()
			mockCache := mocks.NewMockOrderCache()
			mockExtAPI := mocks.NewMockExternalAPIClient()
			tt.setupMock(mockRepo)

			orderSvc := service.NewOrderService(mockRepo, mockCache, mockExtAPI)
			handler := NewOrderHandler(orderSvc)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.UserIDKey), tt.userID)
				c.Next()
			})
			router.GET("/api/orders", handler.ListOrders)

			req := httptest.NewRequest(http.MethodGet, "/api/orders"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
		})
	}
}
