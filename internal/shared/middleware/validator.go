package middleware

import (
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateJSON(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(obj); err != nil {
			response.BadRequest(c, getValidationErrorMessage(err))
			c.Abort()
			return
		}
		c.Next()
	}
}

func ValidateQuery(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindQuery(obj); err != nil {
			response.BadRequest(c, getValidationErrorMessage(err))
			c.Abort()
			return
		}
		c.Next()
	}
}

func ValidateURI(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindUri(obj); err != nil {
			response.BadRequest(c, getValidationErrorMessage(err))
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidationMessage turns binding/validation errors into a short user-facing string (toast-friendly).
func ValidationMessage(err error) string {
	return getValidationErrorMessage(err)
}

func getValidationErrorMessage(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		messages := make([]string, 0, len(validationErrors))
		for _, e := range validationErrors {
			messages = append(messages, getFieldErrorMessage(e))
		}
		return messages[0]
	}
	return "Validation failed"
}

func getFieldErrorMessage(fe validator.FieldError) string {
	field := friendlyFieldName(fe.Field())
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email"
	case "min":
		return field + " must be at least " + fe.Param() + " characters"
	case "max":
		return field + " must be at most " + fe.Param() + " characters"
	case "len":
		return field + " must be exactly " + fe.Param() + " characters"
	case "numeric":
		return field + " must contain only numbers"
	default:
		return field + " is invalid"
	}
}

func friendlyFieldName(field string) string {
	switch field {
	case "OTPCode":
		return "Code"
	case "RefreshToken":
		return "Refresh token"
	case "Name":
		return "Name"
	case "Email":
		return "Email"
	default:
		return field
	}
}
