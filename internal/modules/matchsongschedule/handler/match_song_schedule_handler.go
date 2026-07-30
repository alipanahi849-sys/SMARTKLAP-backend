package handler

import (
	"clap/internal/modules/matchsongschedule/dto"
	"clap/internal/modules/matchsongschedule/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchSongScheduleHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	ListByMatchID(c *gin.Context)
	ListBySongID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type matchSongScheduleHandler struct {
	scheduleService service.MatchSongScheduleService
}

func NewMatchSongScheduleHandler(scheduleService service.MatchSongScheduleService) MatchSongScheduleHandler {
	return &matchSongScheduleHandler{scheduleService: scheduleService}
}

// Create match song schedule godoc
//
//	@Summary		Create match song schedule
//	@Tags			match-song-schedules
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateMatchSongScheduleRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/match-song-schedules [post]
func (h *matchSongScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateMatchSongScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	schedule, err := h.scheduleService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, schedule)
}

// Get match song schedule godoc
//
//	@Summary		Get match song schedule
//	@Tags			match-song-schedules
//	@Produce		json
//	@Param			id	path	string	true	"Schedule ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/match-song-schedules/{id} [get]
func (h *matchSongScheduleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	schedule, err := h.scheduleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedule)
}

// List match song schedules godoc
//
//	@Summary		List match song schedules
//	@Tags			match-song-schedules
//	@Produce		json
//	@Param			sort	query	string	false	"Sort field"
//	@Param			order	query	string	false	"asc|desc"
//	@Param			page	query	int	false	"Page number"
//	@Param			per_page	query	int	false	"Items per page"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/match-song-schedules [get]
func (h *matchSongScheduleHandler) List(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	sortBy := c.Query("sort")
	sortOrder := c.Query("order")

	schedules, err := h.scheduleService.List(c.Request.Context(), page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedules)
}

// List schedules by match godoc
//
//	@Summary		List schedules by match
//	@Tags			match-song-schedules
//	@Produce		json
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/song-schedules [get]
func (h *matchSongScheduleHandler) ListByMatchID(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	schedules, err := h.scheduleService.ListByMatchID(c.Request.Context(), matchID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedules)
}

// List schedules by song godoc
//
//	@Summary		List schedules by song
//	@Tags			match-song-schedules
//	@Produce		json
//	@Param			id	path	string	true	"Song ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id}/match-schedules [get]
func (h *matchSongScheduleHandler) ListBySongID(c *gin.Context) {
	songID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid song ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	schedules, err := h.scheduleService.ListBySongID(c.Request.Context(), songID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedules)
}

// Update match song schedule godoc
//
//	@Summary		Update match song schedule
//	@Tags			match-song-schedules
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Schedule ID"
//	@Param			body	body		dto.UpdateMatchSongScheduleRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/match-song-schedules/{id} [put]
func (h *matchSongScheduleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateMatchSongScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	schedule, err := h.scheduleService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedule)
}

// Delete match song schedule godoc
//
//	@Summary		Delete match song schedule
//	@Tags			match-song-schedules
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Schedule ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/match-song-schedules/{id} [delete]
func (h *matchSongScheduleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.scheduleService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Match song schedule deleted successfully")
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
