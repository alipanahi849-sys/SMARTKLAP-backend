package response

import (
	"encoding/json"
	"net/http"

	"clap/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

// EmptyObject encodes as {} and is not dropped by omitempty.
var EmptyObject = json.RawMessage(`{}`)


type Response struct {
	Success bool         `json:"success"`
	Status  int          `json:"status"`
	Message string       `json:"message"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

type ErrorDetail struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

type PaginatedResponse struct {
	Success bool           `json:"success"`
	Status  int            `json:"status"`
	Message string         `json:"message"`
	Data    interface{}    `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Success(c *gin.Context, data interface{}) {
	SuccessWithMessage(c, data, "OK")
}

func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	if message == "" {
		message = "OK"
	}
	c.JSON(http.StatusOK, Response{
		Success: true,
		Status:  http.StatusOK,
		Message: message,
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	CreatedWithMessage(c, data, "Resource created successfully")
}

func CreatedWithMessage(c *gin.Context, data interface{}, message string) {
	if message == "" {
		message = "Resource created successfully"
	}
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Status:  http.StatusCreated,
		Message: message,
		Data:    data,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, err error) {
	var appErr *errors.AppError
	if customErr, ok := err.(*errors.AppError); ok {
		appErr = customErr
	} else {
		appErr = errors.NewInternal("Internal server error", err)
	}

	c.JSON(appErr.StatusCode, Response{
		Success: false,
		Status:  appErr.StatusCode,
		Message: appErr.Message,
		Error: &ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

func BadRequest(c *gin.Context, message string) {
	writeError(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	writeError(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	writeError(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	writeError(c, http.StatusNotFound, message)
}

func TooManyRequests(c *gin.Context, message string) {
	writeError(c, http.StatusTooManyRequests, message)
}

func RequestTimeout(c *gin.Context, message string) {
	writeError(c, http.StatusRequestTimeout, message)
}

func UnprocessableEntity(c *gin.Context, message string) {
	writeError(c, http.StatusUnprocessableEntity, message)
}

func Paginated(c *gin.Context, data interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "OK",
		Data:    data,
		Meta:    meta,
	})
}

func writeError(c *gin.Context, status int, message string) {
	if message == "" {
		message = http.StatusText(status)
	}
	c.JSON(status, Response{
		Success: false,
		Status:  status,
		Message: message,
		Error: &ErrorDetail{
			Code:    status,
			Message: message,
		},
	})
}
