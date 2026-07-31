package handler

import (
	"clap/internal/modules/realtime/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecoveryHandler serves the reconnection-recovery endpoint.
type RecoveryHandler struct {
	svc service.ReconnectionRecoveryService
}

// NewRecoveryHandler constructs the handler.
func NewRecoveryHandler(svc service.ReconnectionRecoveryService) *RecoveryHandler {
	return &RecoveryHandler{svc: svc}
}

// GetMatchState returns the complete current realtime state for a match.
// Clients call this immediately after reconnecting to re-sync.
//
// GET /api/v1/realtime/session/:matchId
// Realtime session recovery godoc
//
//	@Summary		Realtime session recovery
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Param			matchId	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/session/{matchId} [get]
func (h *RecoveryHandler) GetMatchState(c *gin.Context) {
	matchID, err := uuid.Parse(c.Param("matchId"))
	if err != nil {
		response.BadRequest(c, "invalid match_id")
		return
	}

	state, err := h.svc.GetMatchState(c.Request.Context(), matchID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, state)
}
