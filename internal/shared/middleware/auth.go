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
	TokenTypeKey = "token_type"
)

func Auth() gin.HandlerFunc {
	return authWithAllowedTypes(utils.TokenTypeApp)
}

// AdminAuth accepts only admin-panel JWTs (staff identity, not app users).
func AdminAuth() gin.HandlerFunc {
	return authWithAllowedTypes(utils.TokenTypeAdmin)
}

// AnyAuth accepts either an app or admin access token.
// Use for read endpoints shared by the mobile app and the admin dashboard.
func AnyAuth() gin.HandlerFunc {
	return authWithAllowedTypes(utils.TokenTypeApp, utils.TokenTypeAdmin)
}

func authWithAllowedTypes(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, t := range allowed {
		allowedSet[t] = struct{}{}
	}

	return func(c *gin.Context) {
		claims, ok := parseBearerClaims(c)
		if !ok {
			return
		}

		if _, ok := allowedSet[claims.TokenType]; !ok {
			response.Unauthorized(c, "Invalid token type for this endpoint")
			c.Abort()
			return
		}

		setAuthContext(c, claims)
		c.Next()
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

		claims, err := utils.ValidateAccessToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		setAuthContext(c, claims)
		c.Next()
	}
}

func parseBearerClaims(c *gin.Context) (*utils.Claims, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c, "Authorization header is required")
		c.Abort()
		return nil, false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Unauthorized(c, "Invalid authorization header format")
		c.Abort()
		return nil, false
	}

	claims, err := utils.ValidateAccessToken(parts[1])
	if err != nil {
		response.Error(c, errors.ErrInvalidToken)
		c.Abort()
		return nil, false
	}

	return claims, true
}

func setAuthContext(c *gin.Context, claims *utils.Claims) {
	c.Set(UserIDKey, claims.UserID)
	c.Set(UserEmailKey, claims.Email)
	c.Set(UserRolesKey, claims.Roles)
	c.Set(UserPermsKey, claims.PanelPermissions)
	c.Set(TokenTypeKey, claims.TokenType)
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

func GetTokenType(c *gin.Context) string {
	tokenType, exists := c.Get(TokenTypeKey)
	if !exists {
		return ""
	}
	return tokenType.(string)
}

func IsAdminToken(c *gin.Context) bool {
	return GetTokenType(c) == utils.TokenTypeAdmin
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
