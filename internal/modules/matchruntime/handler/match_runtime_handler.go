package handler

import (
	"clap/internal/modules/matchruntime/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchRuntimeHandler interface {
	Start(c *gin.Context)
	Pause(c *gin.Context)
	Resume(c *gin.Context)
	End(c *gin.Context)
	GetState(c *gin.Context)
}

type matchRuntimeHandler struct {
	svc service.MatchRuntimeService
}

func NewMatchRuntimeHandler(svc service.MatchRuntimeService) MatchRuntimeHandler {
	return &matchRuntimeHandler{svc: svc}
}

// POST /api/v1/matches/:id/runtime/start
// Start match runtime godoc
//
//	@Summary		Start match runtime
//	@Tags			match-runtime
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/runtime/start [post]
func (h *matchRuntimeHandler) Start(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.StartMatch(c.Request.Context(), matchID, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// POST /api/v1/matches/:id/runtime/pause
// Pause match runtime godoc
//
//	@Summary		Pause match runtime
//	@Tags			match-runtime
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/runtime/pause [post]
func (h *matchRuntimeHandler) Pause(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.PauseMatch(c.Request.Context(), matchID, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// POST /api/v1/matches/:id/runtime/resume
// Resume match runtime godoc
//
//	@Summary		Resume match runtime
//	@Tags			match-runtime
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/runtime/resume [post]
func (h *matchRuntimeHandler) Resume(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.ResumeMatch(c.Request.Context(), matchID, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// POST /api/v1/matches/:id/runtime/end
// End match runtime godoc
//
//	@Summary		End match runtime
//	@Tags			match-runtime
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/runtime/end [post]
func (h *matchRuntimeHandler) End(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.EndMatch(c.Request.Context(), matchID, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GET /api/v1/matches/:id/runtime
// Get match runtime state godoc
//
//	@Summary		Get match runtime state
//	@Tags			match-runtime
//	@Produce		json
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id}/runtime [get]
func (h *matchRuntimeHandler) GetState(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}

	result, err := h.svc.GetState(c.Request.Context(), matchID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func getAuthContext(c *gin.Context) *utils.AuthorizationContext {
	userID := middleware.GetUserID(c)
	roles := middleware.GetUserRoles(c)

	var clubID *uuid.UUID
	if s := c.GetHeader("X-Club-ID"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			clubID = &id
		}
	}
	return utils.NewAuthorizationContext(userID, roles, clubID)
}
