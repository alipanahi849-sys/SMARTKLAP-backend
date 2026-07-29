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

func (h *newsHandler) List(c *gin.Context) {
	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.List(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
