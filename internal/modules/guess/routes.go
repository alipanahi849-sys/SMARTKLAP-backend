package guess

import (
	"clap/internal/modules/guess/handler"
	guessrepo "clap/internal/modules/guess/repository"
	"clap/internal/modules/guess/service"
	matchrepo "clap/internal/modules/match/repository"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewGuessHandler(NewService())

	guessGroup := r.Group("/guess")
	guessGroup.Use(middleware.Auth())
	{
		guessGroup.GET("/matches/:match_id", h.MatchOverview)
		guessGroup.GET("/quizzes/:quiz_id", h.QuizDetail)
		guessGroup.POST("/quizzes/:quiz_id/answer", h.Answer)
	}
}

func NewService() service.GuessService {
	db := database.GetDB()
	return service.NewGuessService(
		guessrepo.NewGuessRepository(db),
		matchrepo.NewMatchRepository(db),
		matchrepo.NewMatchDetailsRepository(db),
		settingsrepo.NewSettingsRepository(db),
	)
}
