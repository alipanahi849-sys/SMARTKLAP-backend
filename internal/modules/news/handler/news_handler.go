package handler

import (
	"strings"

	"clap/internal/modules/news/dto"
	"clap/internal/modules/news/service"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type NewsHandler interface {
	List(c *gin.Context)
	GetByID(c *gin.Context)
	GetNewsClub(c *gin.Context)
	SetNewsClub(c *gin.Context)
}

type newsHandler struct {
	svc service.NewsService
}

func NewNewsHandler(svc service.NewsService) NewsHandler {
	return &newsHandler{svc: svc}
}

// List news godoc
//
//	@Summary		List club news
//	@Tags			news
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"Opaque page cursor from the previous response"
//	@Param			limit	query	int		false	"Items per page (default 20, max 100)"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Router			/api/v1/news [get]
func (h *newsHandler) List(c *gin.Context) {
	result, err := h.svc.List(c.Request.Context(), dto.NewsListFilters{
		Cursor: strings.TrimSpace(c.Query("cursor")),
		Limit:  utils.GetMobileCursorLimit(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	if result.Items == nil {
		result.Items = []dto.NewsItem{}
	}
	response.Success(c, result)
}

// GetByID godoc
//
//	@Summary		News article
//	@Tags			news
//	@Produce		json
//	@Security		BearerAuth
//	@Param			news_id	path	string	true	"News ID"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		404		{object}	response.Response
//	@Router			/api/v1/news/{news_id} [get]
func (h *newsHandler) GetByID(c *gin.Context) {
	result, svcErr := h.svc.GetByID(c.Request.Context(), strings.TrimSpace(c.Param("news_id")))
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// GetNewsClub godoc
//
//	@Summary		Get news club
//	@Tags			admin-news
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/settings/news-club [get]
func (h *newsHandler) GetNewsClub(c *gin.Context) {
	result, err := h.svc.GetNewsClub(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// SetNewsClub godoc
//
//	@Summary		Set news club
//	@Tags			admin-news
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.SetNewsClubRequest	true	"News club"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		403		{object}	response.Response
//	@Router			/api/v1/admin/settings/news-club [put]
func (h *newsHandler) SetNewsClub(c *gin.Context) {
	var req dto.SetNewsClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.SetNewsClub(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, result, "News club updated")
}
