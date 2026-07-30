package handler

import (
	"clap/internal/modules/realtime/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// RetentionHandler exposes admin-only data retention cleanup endpoints (CR-14).
type RetentionHandler struct {
	retention service.DataRetentionService
	heartbeat service.HeartbeatCleanupService
}

// NewRetentionHandler constructs the handler.
func NewRetentionHandler(
	retention service.DataRetentionService,
	heartbeat service.HeartbeatCleanupService,
) *RetentionHandler {
	return &RetentionHandler{retention: retention, heartbeat: heartbeat}
}

// CleanupSchedulerEvents deletes terminal scheduler_events beyond retention.
// POST /api/v1/realtime/admin/cleanup/scheduler-events
// Cleanup scheduler events godoc
//
//	@Summary		Cleanup scheduler events
//	@Description	Admin only
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/admin/cleanup/scheduler-events [post]
func (h *RetentionHandler) CleanupSchedulerEvents(c *gin.Context) {
	deleted, err := h.retention.CleanupSchedulerEvents(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": deleted})
}

// CleanupRealtimeEvents deletes realtime_events beyond retention.
// POST /api/v1/realtime/admin/cleanup/realtime-events
// Cleanup realtime events godoc
//
//	@Summary		Cleanup realtime events
//	@Description	Admin only
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/admin/cleanup/realtime-events [post]
func (h *RetentionHandler) CleanupRealtimeEvents(c *gin.Context) {
	deleted, err := h.retention.CleanupRealtimeEvents(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": deleted})
}

// CleanupHeartbeats deletes client_heartbeats beyond retention.
// POST /api/v1/realtime/admin/cleanup/heartbeats
// Cleanup heartbeats godoc
//
//	@Summary		Cleanup heartbeats
//	@Description	Admin only
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/admin/cleanup/heartbeats [post]
func (h *RetentionHandler) CleanupHeartbeats(c *gin.Context) {
	deleted, err := h.heartbeat.Cleanup(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": deleted})
}

// CleanupAll runs both event retention passes.
// POST /api/v1/realtime/admin/cleanup/all
// Cleanup all retention data godoc
//
//	@Summary		Cleanup all retention data
//	@Description	Admin only
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/admin/cleanup/all [post]
func (h *RetentionHandler) CleanupAll(c *gin.Context) {
	result, err := h.retention.CleanupAll(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
