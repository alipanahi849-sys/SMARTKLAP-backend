package handler

import (
	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductHandler serves the mobile Shop screens (contract §6–§7).
type ProductHandler interface {
	List(c *gin.Context)
	GetByID(c *gin.Context)
	Create(c *gin.Context)
	UploadImage(c *gin.Context)
}

type productHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) ProductHandler {
	return &productHandler{svc: svc}
}

// List shop products godoc
//
//	@Summary		List shop products
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search			query	string	false	"Search by name, subname or description"
//	@Param			product_type	query	string	false	"Product type filter (food or merch)"
//	@Param			category		query	string	false	"Category filter"
//	@Param			currency		query	string	false	"Price display currency (EUR or POINT)"
//	@Param			page		query	int		false	"Page number"
//	@Param			limit		query	int		false	"Items per page"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop [get]
func (h *productHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	filters := dto.ProductListFilters{
		Search:      c.Query("search"),
		Category:    c.Query("category"),
		Currency:    c.DefaultQuery("currency", dto.CurrencyEUR),
		ProductType: c.Query("product_type"),
	}

	result, err := h.svc.List(c.Request.Context(), userID, page, limit, filters)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Get shop product godoc
//
//	@Summary		Get shop product
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path	string	true	"Product ID"
//	@Param			currency	query	string	false	"Price display currency (EUR or POINT)"
//	@Param			size		query	string	false	"Merch only: filter available size (M, L, XL, XXL)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop/{id} [get]
func (h *productHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	filters := dto.ProductDetailFilters{
		Currency: c.DefaultQuery("currency", dto.CurrencyEUR),
		Size:     c.Query("size"),
	}

	result, err := h.svc.GetByID(c.Request.Context(), id, filters)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create shop product godoc
//
//	@Summary		Create shop product
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateProductRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop [post]
func (h *productHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// Upload shop image godoc
//
//	@Summary		Upload shop product image
//	@Tags			shop
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Product image (JPEG, PNG or WebP)"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop/image [post]
func (h *productHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.UploadImage(c.Request.Context(), file, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

func getAuthContext(c *gin.Context) *utils.AuthorizationContext {
	return utils.NewAuthorizationContext(
		middleware.GetUserID(c),
		middleware.GetUserRoles(c),
		nil,
	)
}
