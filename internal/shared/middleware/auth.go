package middleware

import (
	"strings"

	"clap/internal/shared/errors"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	UserIDKey    = "user_id"
	UserEmailKey = "user_email"
	UserRolesKey = "user_roles"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateAccessToken(token)
		if err != nil {
			response.Error(c, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRolesKey, claims.Roles)

		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil
	}
	return userID.(uuid.UUID)
}

func GetUserEmail(c *gin.Context) string {
	email, exists := c.Get(UserEmailKey)
	if !exists {
		return ""
	}
	return email.(string)
}

func GetUserRoles(c *gin.Context) []string {
	roles, exists := c.Get(UserRolesKey)
	if !exists {
		return []string{}
	}
	return roles.([]string)
}

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := GetUserRoles(c)

		for _, allowedRole := range allowedRoles {
			for _, userRole := range userRoles {
				if userRole == allowedRole {
					c.Next()
					return
				}
			}
		}

		response.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRolesKey, claims.Roles)

		c.Next()
	}
}
