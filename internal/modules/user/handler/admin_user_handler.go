package handler

import (
	"strconv"
	"strings"

	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminUserHandler exposes admin user CRUD endpoints.
type AdminUserHandler struct {
	svc service.AdminUserService
}

// NewAdminUserHandler constructs the handler.
func NewAdminUserHandler(svc service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{svc: svc}
}

// List godoc
//
//	@Summary		List users (admin)
//	@Tags			admin-users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			q			query	string	false	"Search name/email"
//	@Param			is_active	query	bool	false	"Filter by active"
//	@Param			role		query	string	false	"Filter by role name"
//	@Param			page		query	int		false	"Page (default 1)"
//	@Param			per_page	query	int		false	"Per page (default 20)"
//	@Success		200	{object}	response.Response
//	@Router			/admin/users [get]
func (h *AdminUserHandler) List(c *gin.Context) {
	filters := dto.AdminUserListFilters{
		Query:   strings.TrimSpace(c.Query("q")),
		Role:    strings.TrimSpace(c.Query("role")),
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "per_page", 20),
	}
	if raw := strings.TrimSpace(c.Query("is_active")); raw != "" {
		active := raw == "1" || strings.EqualFold(raw, "true")
		filters.IsActive = &active
	}

	result, err := h.svc.List(c.Request.Context(), filters)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Get godoc
//
//	@Summary		Get user (admin)
//	@Tags			admin-users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"User ID"
//	@Success		200	{object}	response.Response
//	@Router			/admin/users/{id} [get]
func (h *AdminUserHandler) Get(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user id")
		return
	}
	result, err := h.svc.Get(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create godoc
//
//	@Summary		Create panel user (admin)
//	@Tags			admin-users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	dto.AdminCreateUserRequest	true	"New user"
//	@Success		200		{object}	response.Response
//	@Router			/admin/users [post]
func (h *AdminUserHandler) Create(c *gin.Context) {
	var req dto.AdminCreateUserRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.Create(c.Request.Context(), middleware.GetUserID(c), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Update godoc
//
//	@Summary		Update user (admin)
//	@Tags			admin-users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"User ID"
//	@Param			body	body	dto.AdminUpdateUserRequest	true	"Fields to update"
//	@Success		200		{object}	response.Response
//	@Router			/admin/users/{id} [patch]
func (h *AdminUserHandler) Update(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user id")
		return
	}

	var req dto.AdminUpdateUserRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	result, err := h.svc.Update(c.Request.Context(), middleware.GetUserID(c), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
