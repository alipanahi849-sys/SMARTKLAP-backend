package handler

import (
	"strings"

	"clap/internal/modules/news/dto"
	"clap/internal/modules/news/service"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NewsHandler interface {
	List(c *gin.Context)
	GetByID(c *gin.Context)
}

type newsHandler struct {
	svc service.NewsService
}

func NewNewsHandler(svc service.NewsService) NewsHandler {
	return &newsHandler{svc: svc}
}

// List news godoc
//
//	@Summary		List news
//	@Tags			news
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"News ID of the last item from the previous page"
//	@Param			limit	query	int		false	"Items per page (default 20, max 100)"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Router			/api/v1/news [get]
func (h *newsHandler) List(c *gin.Context) {
	limit := utils.GetMobileCursorLimit(c)

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	result, err := h.svc.List(c.Request.Context(), dto.NewsListFilters{
		Cursor: cursor,
		Limit:  limit,
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
	id, err := uuid.Parse(c.Param("news_id"))
	if err != nil {
		response.BadRequest(c, "Invalid news ID")
		return
	}
	result, svcErr := h.svc.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
