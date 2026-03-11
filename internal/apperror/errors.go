package apperror

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error codes.
type ErrorCode string

const (
	// Authentication errors
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       ErrorCode = "TOKEN_INVALID"

	// Validation errors
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// Resource errors
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeConflict    ErrorCode = "CONFLICT"
	ErrCodeUserExists  ErrorCode = "USER_EXISTS"

	// Server errors
	ErrCodeInternal    ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// Rate limiting
	ErrCodeRateLimited ErrorCode = "RATE_LIMITED"
)

// AppError represents a structured application error.
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	HTTPStatus int       `json:"-"`
	Err        error     `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// ErrorResponse is the JSON response format for errors.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains error details for JSON response.
type ErrorBody struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

// ToResponse converts AppError to ErrorResponse.
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorBody{
			Code:    e.Code,
			Message: e.Message,
			Detail:  e.Detail,
		},
	}
}

// Constructor functions for common errors

// NewUnauthorized creates an unauthorized error.
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewInvalidCredentials creates an invalid credentials error.
func NewInvalidCredentials() *AppError {
	return &AppError{
		Code:       ErrCodeInvalidCredentials,
		Message:    "Invalid email or password",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewValidation creates a validation error.
func NewValidation(message string) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewNotFound creates a not found error.
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewConflict creates a conflict error.
func NewConflict(message string) *AppError {
	return &AppError{
		Code:       ErrCodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewUserExists creates a user exists error.
func NewUserExists() *AppError {
	return &AppError{
		Code:       ErrCodeUserExists,
		Message:    "User with this email already exists",
		HTTPStatus: http.StatusConflict,
	}
}

// NewInternal creates an internal server error.
func NewInternal(message string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewRateLimited creates a rate limited error.
func NewRateLimited() *AppError {
	return &AppError{
		Code:       ErrCodeRateLimited,
		Message:    "Too many requests, please try again later",
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// NewTokenExpired creates a token expired error.
func NewTokenExpired() *AppError {
	return &AppError{
		Code:       ErrCodeTokenExpired,
		Message:    "Token has expired",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewTokenInvalid creates an invalid token error.
func NewTokenInvalid() *AppError {
	return &AppError{
		Code:       ErrCodeTokenInvalid,
		Message:    "Invalid or malformed token",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// Is checks if an error matches an AppError code.
func Is(err error, code ErrorCode) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}
