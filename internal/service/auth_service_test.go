package service

import (
	"context"
	"errors"
	"order-management-api/internal/domain"
	"order-management-api/internal/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key-for-testing-purposes-32-chars"
	jwtExpiration := 24 * time.Hour

	tests := []struct {
		name          string
		input         RegisterInput
		setupMock     func(*mocks.MockUserRepository)
		expectedError error
		checkResult   func(*testing.T, *AuthResponse)
	}{
		{
			name: "successful registration",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "test@example.com").Return(nil, gorm.ErrRecordNotFound)
				m.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, resp *AuthResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "test@example.com", resp.User.Email)
				assert.Equal(t, "Test User", resp.User.Name)
				assert.NotEmpty(t, resp.Token)
				assert.NotEmpty(t, resp.User.ID)
			},
		},
		{
			name: "user already exists",
			input: RegisterInput{
				Email:    "existing@example.com",
				Password: "password123",
				Name:     "Existing User",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "existing@example.com").Return(&domain.User{
					ID:    "existing-id",
					Email: "existing@example.com",
				}, nil)
			},
			expectedError: ErrUserExists,
			checkResult:   nil,
		},
		{
			name: "repository create error",
			input: RegisterInput{
				Email:    "new@example.com",
				Password: "password123",
				Name:     "New User",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "new@example.com").Return(nil, gorm.ErrRecordNotFound)
				m.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(errors.New("database error"))
			},
			expectedError: errors.New("database error"),
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository()
			tt.setupMock(mockRepo)

			svc := NewAuthService(mockRepo, jwtSecret, jwtExpiration)
			resp, err := svc.Register(ctx, tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if tt.expectedError.Error() != "" {
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
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

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "test-secret-key-for-testing-purposes-32-chars"
	jwtExpiration := 24 * time.Hour

	// Create a valid password hash
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	tests := []struct {
		name          string
		input         LoginInput
		setupMock     func(*mocks.MockUserRepository)
		expectedError error
		checkResult   func(*testing.T, *AuthResponse)
	}{
		{
			name: "successful login",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "correctpassword",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "test@example.com").Return(&domain.User{
					ID:       "user-id",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
				}, nil)
			},
			expectedError: nil,
			checkResult: func(t *testing.T, resp *AuthResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "test@example.com", resp.User.Email)
				assert.NotEmpty(t, resp.Token)
			},
		},
		{
			name: "user not found",
			input: LoginInput{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "nonexistent@example.com").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrInvalidCredentials,
			checkResult:   nil,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", ctx, "test@example.com").Return(&domain.User{
					ID:       "user-id",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
				}, nil)
			},
			expectedError: ErrInvalidCredentials,
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository()
			tt.setupMock(mockRepo)

			svc := NewAuthService(mockRepo, jwtSecret, jwtExpiration)
			resp, err := svc.Login(ctx, tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
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
