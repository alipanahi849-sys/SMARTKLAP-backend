package handler

import (
	"net/http"

	"clap/internal/modules/realtime/metrics"

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
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Router			/api/v1/realtime/metrics [get]
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, h.m.Snapshot())
}
