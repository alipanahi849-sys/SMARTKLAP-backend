package handler

import (
	"clap/internal/modules/realtime/dto"
	"clap/internal/modules/realtime/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type TimeSyncHandler interface {
	GetServerTime(c *gin.Context)
	Sync(c *gin.Context)
}

type timeSyncHandler struct {
	svc service.TimeSyncService
}

func NewTimeSyncHandler(svc service.TimeSyncService) TimeSyncHandler {
	return &timeSyncHandler{svc: svc}
}

// GetServerTime returns the authoritative server timestamp.
// GET /realtime/time
func (h *timeSyncHandler) GetServerTime(c *gin.Context) {
	result := h.svc.GetServerTime(c.Request.Context())
	response.Success(c, result)
}

// Sync accepts a client timestamp and returns a full sync payload with drift info.
// GET /realtime/time-sync  (also supports POST for richer request body)
func (h *timeSyncHandler) Sync(c *gin.Context) {
	var req dto.TimeSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "client_timestamp_ms is required")
		return
	}

	result := h.svc.BuildSyncPayload(c.Request.Context(), req.ClientTimestampMs)
	response.Success(c, result)
}
