package handler

import (
	"clap/internal/modules/club/dto"
	"clap/internal/modules/club/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClubHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	Search(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type clubHandler struct {
	clubService service.ClubService
}

func NewClubHandler(clubService service.ClubService) ClubHandler {
	return &clubHandler{clubService: clubService}
}

// Create club godoc
//
//	@Summary		Create club
//	@Tags			clubs
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateClubRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs [post]
func (h *clubHandler) Create(c *gin.Context) {
	var req dto.CreateClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	club, err := h.clubService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, club)
}

// Get club godoc
//
//	@Summary		Get club
//	@Tags			clubs
//	@Produce		json
//	@Param			id	path	string	true	"Club ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs/{id} [get]
func (h *clubHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	club, err := h.clubService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, club)
}

// List clubs godoc
//
//	@Summary		List clubs
//	@Tags			clubs
//	@Produce		json
//	@Param			sort	query	string	false	"Sort field"
//	@Param			order	query	string	false	"asc|desc"
//	@Param			page	query	int	false	"Page number"
//	@Param			per_page	query	int	false	"Items per page"
//	@Param			q	query	string	false	"Search query"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs [get]
func (h *clubHandler) List(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	sortBy := c.Query("sort")
	sortOrder := c.Query("order")

	clubs, err := h.clubService.List(c.Request.Context(), page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, clubs)
}

// Search clubs godoc
//
//	@Summary		Search clubs
//	@Tags			clubs
//	@Produce		json
//	@Param			q	query	string	true	"Search query"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs/search [get]
func (h *clubHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "Search query is required")
		return
	}

	page, pageSize := utils.GetPagination(c)

	clubs, err := h.clubService.Search(c.Request.Context(), query, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, clubs)
}

// Update club godoc
//
//	@Summary		Update club
//	@Tags			clubs
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Club ID"
//	@Param			body	body		dto.UpdateClubRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs/{id} [put]
func (h *clubHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	club, err := h.clubService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, club)
}

// Delete club godoc
//
//	@Summary		Delete club
//	@Tags			clubs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Club ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs/{id} [delete]
func (h *clubHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.clubService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Club deleted successfully")
}

func getAuthContext(c *gin.Context) *utils.AuthorizationContext {
	userID := middleware.GetUserID(c)
	roles := middleware.GetUserRoles(c)

	var clubID *uuid.UUID
	if clubIDStr := c.GetHeader("X-Club-ID"); clubIDStr != "" {
		if id, err := uuid.Parse(clubIDStr); err == nil {
			clubID = &id
		}
	}

	return utils.NewAuthorizationContext(userID, roles, clubID)
}
