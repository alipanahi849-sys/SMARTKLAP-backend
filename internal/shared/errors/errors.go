package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Err        error  `json:"-"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewBadRequest(message string, err error) *AppError {
	return &AppError{
		Code:       400,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusBadRequest,
	}
}

func NewUnauthorized(message string, err error) *AppError {
	return &AppError{
		Code:       401,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewForbidden(message string, err error) *AppError {
	return &AppError{
		Code:       403,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusForbidden,
	}
}

func NewNotFound(message string, err error) *AppError {
	return &AppError{
		Code:       404,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusNotFound,
	}
}

func NewConflict(message string, err error) *AppError {
	return &AppError{
		Code:       409,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusConflict,
	}
}

// NewUnprocessable returns a 422 business-rule violation (Mobile API Contract
// uses 422 for cases like "quiz already answered" or "cart is empty").
func NewUnprocessable(message string, err error) *AppError {
	return &AppError{
		Code:       422,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusUnprocessableEntity,
	}
}

// NewTooManyRequests returns a 429 rate-limit error.
func NewTooManyRequests(message string, err error) *AppError {
	return &AppError{
		Code:       429,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusTooManyRequests,
	}
}

// NewPayloadTooLarge returns a 413 upload-size error.
func NewPayloadTooLarge(message string, err error) *AppError {
	return &AppError{
		Code:       413,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusRequestEntityTooLarge,
	}
}

// NewUnsupportedMedia returns a 415 media-format error.
func NewUnsupportedMedia(message string, err error) *AppError {
	return &AppError{
		Code:       415,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusUnsupportedMediaType,
	}
}

func NewInternal(message string, err error) *AppError {
	return &AppError{
		Code:       500,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewServiceUnavailable(message string, err error) *AppError {
	return &AppError{
		Code:       503,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusServiceUnavailable,
	}
}

var (
	ErrInvalidCredentials = NewUnauthorized("Invalid credentials", nil)
	ErrInvalidToken       = NewUnauthorized("Invalid or expired token", nil)
	ErrUserNotFound       = NewNotFound("User not found", nil)
	ErrEmailExists        = NewConflict("Email already exists", nil)
	ErrInvalidInput       = NewBadRequest("Invalid input", nil)
	ErrPermissionDenied   = NewForbidden("Permission denied", nil)
)
