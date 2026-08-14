package match

import (
	clubrepo "clap/internal/modules/club/repository"
	leaguerepo "clap/internal/modules/league/repository"
	"clap/internal/modules/match/handler"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/modules/match/service"
	playerrepo "clap/internal/modules/player/repository"
	seasonrepo "clap/internal/modules/season/repository"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/utils"
	"clap/pkg/football"

	"github.com/gin-gonic/gin"
)

func newProvider() football.Provider {
	apiKey := ""
	baseURL := ""
	if config.AppConfig != nil {
		apiKey = config.AppConfig.Football.APIKey
		baseURL = config.AppConfig.Football.BaseURL
	}
	return football.NewAPIFootball(apiKey, baseURL)
}

// NewSyncService wires the football fixture syncer for the API process.
func NewSyncService() *service.SyncService {
	db := database.GetDB()
	return service.NewSyncService(
		newProvider(),
		matchrepo.NewMatchRepository(db),
		matchrepo.NewMatchDetailsRepository(db),
		clubrepo.NewClubRepository(db),
		leaguerepo.NewLeagueRepository(db),
		seasonrepo.NewSeasonRepository(db),
		playerrepo.NewPlayerRepository(db),
		settingsrepo.NewSettingsRepository(db),
	)
}

// RegisterRoutes wires mobile match/player endpoints and admin featured-club APIs.
func RegisterRoutes(r *gin.RouterGroup, syncer *service.SyncService) {
	db := database.GetDB()
	provider := newProvider()
	if syncer == nil {
		syncer = NewSyncService()
	}

	svc := service.NewMatchService(
		matchrepo.NewMatchRepository(db),
		matchrepo.NewMatchDetailsRepository(db),
		clubrepo.NewClubRepository(db),
		playerrepo.NewPlayerRepository(db),
		settingsrepo.NewSettingsRepository(db),
		provider,
		syncer,
	)
	h := handler.NewMatchHandler(svc)

	matches := r.Group("/matches")
	matches.Use(middleware.Auth())
	{
		matches.GET("/current", h.GetCurrent)
		matches.GET("/:match_id", h.GetByID)
	}

	players := r.Group("/players")
	players.Use(middleware.Auth())
	{
		players.GET("/:player_id", h.GetPlayer)
	}

	admin := r.Group("/admin")
	admin.Use(middleware.Auth(), middleware.RequireRole(string(utils.RoleAdmin)))
	{
		admin.GET("/football/teams", h.SearchTeams)
		admin.GET("/settings/featured-club", h.GetFeaturedClub)
		admin.PUT("/settings/featured-club", h.SetFeaturedClub)
		admin.POST("/matches/sync", h.Sync)
	}
}
