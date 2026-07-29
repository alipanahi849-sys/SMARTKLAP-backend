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
	switch fe.Tag() {
	case "required":
		return "Field " + fe.Field() + " is required"
	case "email":
		return "Field " + fe.Field() + " must be a valid email"
	case "min":
		return "Field " + fe.Field() + " must be at least " + fe.Param() + " characters"
	case "max":
		return "Field " + fe.Field() + " must be at most " + fe.Param() + " characters"
	default:
		return "Field " + fe.Field() + " is invalid"
	}
}
