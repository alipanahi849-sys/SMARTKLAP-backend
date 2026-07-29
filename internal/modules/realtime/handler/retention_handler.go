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
func (h *RetentionHandler) CleanupAll(c *gin.Context) {
	result, err := h.retention.CleanupAll(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
