package handler

import (
	"net/http"

	"clap/internal/modules/realtime/metrics"
	"clap/internal/modules/realtime/ws"
	"clap/internal/shared/config"
	"clap/internal/shared/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// upgrader validates the request Origin against the configured allowlist
// before upgrading (CR-8 / F-009). Non-browser clients that send no Origin
// header are permitted; browser origins must match the allowlist.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return config.IsWSOriginAllowed(r.Header.Get("Origin"))
	},
}

// WSHandler manages the HTTP→WebSocket upgrade flow.
type WSHandler struct {
	cm      *ws.ConnectionManager
	metrics *metrics.Metrics
}

// NewWSHandler constructs the handler.
func NewWSHandler(cm *ws.ConnectionManager, m *metrics.Metrics) *WSHandler {
	return &WSHandler{cm: cm, metrics: m}
}

// Connect upgrades an HTTP connection to WebSocket after JWT validation.
// GET /api/v1/realtime/ws
//
// Authentication: Authorization: Bearer <token> (header only).
func (h *WSHandler) Connect(c *gin.Context) {
	// 1. Authenticate before the upgrade — send HTTP 401 on failure.
	auth, err := ws.Authenticate(c.Request)
	if err != nil {
		if h.metrics != nil {
			h.metrics.AuthFailures.Add(1)
		}
		logger.Warn().
			Str("remote_addr", c.Request.RemoteAddr).
			Str("client_ip", c.ClientIP()).
			Err(err).
			Msg("websocket auth failure")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	// 2. Upgrade.
	conn, upgradeErr := upgrader.Upgrade(c.Writer, c.Request, nil)
	if upgradeErr != nil {
		logger.Error().
			Str("user_id", auth.UserID.String()).
			Err(upgradeErr).
			Msg("websocket upgrade failed")
		return
	}

	// 3. Register client (with IP for subscription rate limiting and token
	//    expiry for session-expiry enforcement) and start pumps.
	client := ws.NewClientWithOptions(h.cm.Hub(), conn, auth.UserID, ws.ClientOptions{
		ClientIP:       c.ClientIP(),
		TokenExpiresAt: auth.ExpiresAt,
	})

	go client.WritePump()
	go client.ReadPump()
}
