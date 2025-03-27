package errors

import (
	"errors"
	"fmt"
)

// ErrorCode định nghĩa mã lỗi
type ErrorCode string

const (
	// Auth errors
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken    ErrorCode = "INVALID_TOKEN"
	ErrCodeMissingToken    ErrorCode = "MISSING_TOKEN"
	ErrCodeInvalidPassword ErrorCode = "INVALID_PASSWORD"
	ErrCodeUserNotFound    ErrorCode = "USER_NOT_FOUND"
	ErrCodeUserExists      ErrorCode = "USER_EXISTS"
	ErrCodeInvalidEmail    ErrorCode = "INVALID_EMAIL"
	ErrCodeInvalidPhone    ErrorCode = "INVALID_PHONE"
	ErrCodeInvalidCode     ErrorCode = "INVALID_CODE"
	ErrCodeExpiredCode     ErrorCode = "EXPIRED_CODE"
	ErrCodeInvalidRole     ErrorCode = "INVALID_ROLE"

	// User errors
	ErrCodeInvalidUserID  ErrorCode = "INVALID_USER_ID"
	ErrCodeInvalidStatus  ErrorCode = "INVALID_STATUS"
	ErrCodeInvalidAmount  ErrorCode = "INVALID_AMOUNT"
	ErrCodeInvalidBank    ErrorCode = "INVALID_BANK"
	ErrCodeInvalidBankID  ErrorCode = "INVALID_BANK_ID"
	ErrCodeBankExists     ErrorCode = "BANK_EXISTS"
	ErrCodeInvalidAccount ErrorCode = "INVALID_ACCOUNT"

	// Database errors
	ErrCodeDBError     ErrorCode = "DB_ERROR"
	ErrCodeDBNotFound  ErrorCode = "DB_NOT_FOUND"
	ErrCodeDBDuplicate ErrorCode = "DB_DUPLICATE"

	// Validation errors
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeRequiredField ErrorCode = "REQUIRED_FIELD"
	ErrCodeInvalidFormat ErrorCode = "INVALID_FORMAT"

	// Business errors
	ErrCodeInsufficientFund ErrorCode = "INSUFFICIENT_FUND"
	ErrCodeInvalidOperation ErrorCode = "INVALID_OPERATION"
)

// AppError định nghĩa lỗi của ứng dụng
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError tạo một AppError mới
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsAppError kiểm tra xem error có phải là AppError không
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError lấy AppError từ error
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUnauthorized      = errors.New("unauthorized")

	// Order errors
	ErrOrderNotFound     = errors.New("order not found")
	ErrOrderInvalid      = errors.New("invalid order")
	ErrOrderCancelled    = errors.New("order already cancelled")
	ErrOrderCompleted    = errors.New("order already completed")
	ErrOrderNotConfirmed = errors.New("order not confirmed")

	// Room errors
	ErrRoomNotFound     = errors.New("room not found")
	ErrRoomNotAvailable = errors.New("room not available")
	ErrRoomOccupied     = errors.New("room is occupied")

	// Payment errors
	ErrPaymentFailed   = errors.New("payment failed")
	ErrPaymentRefunded = errors.New("payment already refunded")
	ErrInvalidAmount   = errors.New("invalid amount")

	// Validation errors
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingRequired = errors.New("missing required field")
	ErrInvalidFormat   = errors.New("invalid format")
)
