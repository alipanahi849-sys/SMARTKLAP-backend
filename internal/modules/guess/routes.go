package guess

import (
	"clap/internal/modules/guess/handler"
	"clap/internal/modules/guess/repository"
	"clap/internal/modules/guess/service"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Guess endpoints (Mobile API Contract §5).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	guessSvc := service.NewGuessService(
		repository.NewQuizRepository(db),
		matchrepo.NewMatchRepository(db),
	)
	h := handler.NewGuessHandler(guessSvc)

	guessGroup := r.Group("/guess")
	guessGroup.Use(middleware.Auth())
	{
		guessGroup.GET("/matches/:match_id", h.MatchOverview)
		guessGroup.GET("/quizzes/:quiz_id", h.QuizDetail)
		guessGroup.POST("/quizzes/:quiz_id/answer", h.Answer)
	}
}
