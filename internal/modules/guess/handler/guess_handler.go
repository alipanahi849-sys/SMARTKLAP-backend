package handler

import (
	"clap/internal/modules/guess/dto"
	"clap/internal/modules/guess/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GuessHandler interface {
	MatchOverview(c *gin.Context)
	QuizDetail(c *gin.Context)
	Answer(c *gin.Context)
}

type guessHandler struct {
	svc service.GuessService
}

func NewGuessHandler(svc service.GuessService) GuessHandler {
	return &guessHandler{svc: svc}
}

// MatchOverview godoc
//
//	@Summary		Guess match overview
//	@Tags			guess
//	@Produce		json
//	@Security		BearerAuth
//	@Param			match_id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/guess/matches/{match_id} [get]
func (h *guessHandler) MatchOverview(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	result, err := h.svc.MatchOverview(c.Request.Context(), userID, c.Param("match_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// QuizDetail godoc
//
//	@Summary		Quiz detail
//	@Tags			guess
//	@Produce		json
//	@Security		BearerAuth
//	@Param			quiz_id	path	string	true	"Quiz ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/guess/quizzes/{quiz_id} [get]
func (h *guessHandler) QuizDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}
	quizID, err := uuid.Parse(c.Param("quiz_id"))
	if err != nil {
		response.BadRequest(c, "Invalid quiz ID")
		return
	}

	result, svcErr := h.svc.QuizDetail(c.Request.Context(), userID, quizID)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// Answer godoc
//
//	@Summary		Answer quiz
//	@Tags			guess
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			quiz_id	path	string	true	"Quiz ID"
//	@Param			body	body	dto.AnswerQuizRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		409	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/guess/quizzes/{quiz_id}/answer [post]
func (h *guessHandler) Answer(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}
	quizID, err := uuid.Parse(c.Param("quiz_id"))
	if err != nil {
		response.BadRequest(c, "Invalid quiz ID")
		return
	}

	var req dto.AnswerQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, svcErr := h.svc.Answer(c.Request.Context(), userID, quizID, &req)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
