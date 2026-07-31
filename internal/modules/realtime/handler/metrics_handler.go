package handler

import (
	"clap/internal/modules/realtime/metrics"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// MetricsHandler exposes internal WebSocket metrics.
type MetricsHandler struct {
	m *metrics.Metrics
}

// NewMetricsHandler constructs the handler.
func NewMetricsHandler(m *metrics.Metrics) *MetricsHandler {
	return &MetricsHandler{m: m}
}

// GetMetrics returns a snapshot of current WebSocket metrics.
// GET /api/v1/realtime/metrics
// Realtime metrics godoc
//
//	@Summary		Realtime metrics
//	@Description	Admin only
//	@Tags			realtime
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/realtime/metrics [get]
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	response.Success(c, h.m.Snapshot())
}
