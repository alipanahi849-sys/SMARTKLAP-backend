package response

import (
	"net/http"

	"clap/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool         `json:"success"`
	Data    interface{}  `json:"data,omitempty"`
	Message string       `json:"message,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

type ErrorDetail struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

type PaginatedResponse struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data"`
	Message string         `json:"message,omitempty"`
	Meta    PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
		Message: "Resource created successfully",
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
		Error: &ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    400,
			Message: message,
		},
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    401,
			Message: message,
		},
	})
}

func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    403,
			Message: message,
		},
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    404,
			Message: message,
		},
	})
}

func Paginated(c *gin.Context, data interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}
