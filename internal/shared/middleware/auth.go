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
	UserPermsKey = "panel_permissions"
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
		c.Set(UserPermsKey, claims.PanelPermissions)

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

func GetPanelPermissions(c *gin.Context) []string {
	perms, exists := c.Get(UserPermsKey)
	if !exists || perms == nil {
		return []string{}
	}
	switch v := perms.(type) {
	case []string:
		return v
	default:
		return []string{}
	}
}

func hasRole(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}

func hasPermission(perms []string, want string) bool {
	for _, perm := range perms {
		if perm == want {
			return true
		}
	}
	return false
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

// RequirePermission allows full admins, or users granted the given panel section.
func RequirePermission(section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasRole(GetUserRoles(c), string(utils.RoleAdmin)) {
			c.Next()
			return
		}
		if hasPermission(GetPanelPermissions(c), section) {
			c.Next()
			return
		}
		response.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

// RequirePanelAccess allows full admins or anyone with at least one panel section.
func RequirePanelAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasRole(GetUserRoles(c), string(utils.RoleAdmin)) {
			c.Next()
			return
		}
		if len(GetPanelPermissions(c)) > 0 {
			c.Next()
			return
		}
		response.Forbidden(c, "Admin panel access required")
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
		c.Set(UserPermsKey, claims.PanelPermissions)

		c.Next()
	}
}
