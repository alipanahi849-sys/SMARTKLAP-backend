package matchruntime

import (
	"clap/internal/modules/matchruntime/handler"
	"clap/internal/modules/matchruntime/repository"
	"clap/internal/modules/matchruntime/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires match runtime endpoints without a realtime publisher.
// Used in environments that do not run the WebSocket layer (e.g. integration tests).
func RegisterRoutes(r *gin.RouterGroup) {
	RegisterRoutesWithPublisher(r, nil)
}

// RegisterRoutesWithPublisher wires match runtime endpoints with an optional
// MatchEventPublisher.  When publisher is non-nil, state-change mutations will
// push match.runtime.updated events to all subscribed WebSocket clients.
func RegisterRoutesWithPublisher(r *gin.RouterGroup, publisher service.MatchEventPublisher) {
	db := database.GetDB()
	repo := repository.NewMatchRuntimeRepository(db)

	var svc service.MatchRuntimeService
	if publisher != nil {
		svc = service.NewMatchRuntimeServiceWithPublisher(repo, service.RealClock(), publisher)
	} else {
		svc = service.NewMatchRuntimeService(repo, service.RealClock())
	}

	h := handler.NewMatchRuntimeHandler(svc)

	matches := r.Group("/matches/:id/runtime")
	{
		auth := matches.Group("")
		auth.Use(middleware.Auth())
		auth.POST("/start", h.Start)
		auth.POST("/pause", h.Pause)
		auth.POST("/resume", h.Resume)
		auth.POST("/end", h.End)

		matches.GET("", h.GetState)
	}
}
