package handler

import (
	"clap/internal/modules/realtime/dto"
	"clap/internal/modules/realtime/gateway"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestEmitHandler lets admins push a one-off event over WebSocket for smoke tests.
type TestEmitHandler struct {
	gw *gateway.WebSocketRealtimeGateway
}

// NewTestEmitHandler constructs the handler.
func NewTestEmitHandler(gw *gateway.WebSocketRealtimeGateway) *TestEmitHandler {
	return &TestEmitHandler{gw: gw}
}

// TestEmitRequest is the body for POST /realtime/admin/test-emit.
type TestEmitRequest struct {
	MatchID *string `json:"match_id"`
	Message string  `json:"message"`
	Broadcast bool  `json:"broadcast"`
}

// Emit pushes a server.notification envelope to a match channel or all clients.
// POST /api/v1/realtime/admin/test-emit
//
//	@Summary		Emit a test WebSocket event
//	@Description	Admin only — smoke-test realtime delivery
//	@Tags			realtime
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		TestEmitRequest	false	"Optional match_id / message / broadcast"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Router			/api/v1/realtime/admin/test-emit [post]
func (h *TestEmitHandler) Emit(c *gin.Context) {
	if h.gw == nil {
		response.BadRequest(c, "realtime gateway is not available")
		return
	}

	var req TestEmitRequest
	_ = c.ShouldBindJSON(&req)

	msg := req.Message
	if msg == "" {
		msg = "socket smoke test"
	}

	payload := map[string]any{
		"message": msg,
		"source":  "admin.test-emit",
	}

	var (
		env     *dto.EventEnvelope
		matchID *uuid.UUID
	)

	if req.MatchID != nil && *req.MatchID != "" {
		parsed, err := uuid.Parse(*req.MatchID)
		if err != nil {
			response.BadRequest(c, "invalid match_id")
			return
		}
		matchID = &parsed
	}

	env = dto.NewEnvelope(dto.EventTypeServerNotification, matchID, payload)

	var err error
	switch {
	case req.Broadcast || matchID == nil:
		err = h.gw.BroadcastEnvelope(c.Request.Context(), env)
	default:
		err = h.gw.PublishToMatch(c.Request.Context(), *matchID, env)
	}
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"emitted":  true,
		"event_id": env.ID,
		"type":     env.Type,
		"match_id": matchID,
		"broadcast": req.Broadcast || matchID == nil,
	})
}
