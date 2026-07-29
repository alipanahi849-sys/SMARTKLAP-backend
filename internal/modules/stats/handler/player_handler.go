package handler

import (
	"clap/internal/modules/stats/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlayerHandler serves GET /players/{player_id} (contract §9.2).
type PlayerHandler interface {
	PlayerDetail(c *gin.Context)
}

type playerHandler struct {
	svc service.PlayerService
}

func NewPlayerHandler(svc service.PlayerService) PlayerHandler {
	return &playerHandler{svc: svc}
}

func (h *playerHandler) PlayerDetail(c *gin.Context) {
	playerID, err := uuid.Parse(c.Param("player_id"))
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	var matchID *uuid.UUID
	if raw, ok := c.GetQuery("match_id"); ok && raw != "" {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			response.BadRequest(c, "Invalid match_id")
			return
		}
		matchID = &parsed
	}

	result, svcErr := h.svc.PlayerDetail(c.Request.Context(), playerID, matchID)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
