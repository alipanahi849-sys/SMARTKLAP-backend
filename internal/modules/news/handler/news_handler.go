package handler

import (
	"clap/internal/modules/news/service"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type NewsHandler interface {
	List(c *gin.Context)
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
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/news [get]
func (h *newsHandler) List(c *gin.Context) {
	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.List(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
