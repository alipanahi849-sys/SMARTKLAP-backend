package dto

import "github.com/google/uuid"

// AdminUserListFilters are query params for GET /api/v1/admin/users.
type AdminUserListFilters struct {
	Query    string
	IsActive *bool
	Role     string
	Page     int
	PerPage  int
}

// AdminUserItem is one row in the admin users list / detail.
type AdminUserItem struct {
	ID               uuid.UUID `json:"id"`
	Email            string    `json:"email"`
	Name             string    `json:"name"`
	Phone            string    `json:"phone,omitempty"`
	Points           int       `json:"points"`
	IsActive         bool      `json:"is_active"`
	IsVerified       bool      `json:"is_verified"`
	Roles            []string  `json:"roles"`
	PanelPermissions []string  `json:"panel_permissions"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at,omitempty"`
}

// AdminUserListResponse is GET /api/v1/admin/users.
type AdminUserListResponse struct {
	Items []AdminUserItem   `json:"items"`
	Meta  AdminUserListMeta `json:"meta"`
}

// AdminUserListMeta is offset pagination for admin users.
type AdminUserListMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// AdminCreateUserRequest is POST /api/v1/admin/users.
type AdminCreateUserRequest struct {
	Email            string   `json:"email" binding:"required"`
	Name             string   `json:"name"`
	Phone            string   `json:"phone"`
	Points           *int     `json:"points"`
	IsActive         *bool    `json:"is_active"`
	Roles            []string `json:"roles"`
	PanelPermissions []string `json:"panel_permissions"`
}

// AdminUpdateUserRequest is PATCH /api/v1/admin/users/:id.
type AdminUpdateUserRequest struct {
	Name             *string  `json:"name"`
	Email            *string  `json:"email"`
	Phone            *string  `json:"phone"`
	Points           *int     `json:"points"`
	IsActive         *bool    `json:"is_active"`
	IsVerified       *bool    `json:"is_verified"`
	Roles            []string `json:"roles"`
	PanelPermissions []string `json:"panel_permissions"`
}
