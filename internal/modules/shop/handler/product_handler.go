package handler

import (
	"strings"

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
	Update(c *gin.Context)
	Delete(c *gin.Context)
	UploadProductImage(c *gin.Context)
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
//	@Param			cursor			query	string	false	"Product ID of the last item from the previous page"
//	@Param			limit			query	int		false	"Items per page (default 20, max 100)"
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

	filters := dto.ProductListFilters{
		Search:      c.Query("search"),
		Category:    c.Query("category"),
		Currency:    c.DefaultQuery("currency", dto.CurrencyEUR),
		ProductType: c.Query("product_type"),
		Cursor:      cursor,
		Limit:       limit,
	}

	result, err := h.svc.List(c.Request.Context(), userID, filters)
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

// Update shop product godoc
//
//	@Summary		Update shop product
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string	true	"Product ID"
//	@Param			body	body	dto.UpdateProductRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop/{id} [put]
func (h *productHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Delete shop product godoc
//
//	@Summary		Delete shop product
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Product ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/shop/{id} [delete]
func (h *productHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.svc.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, nil, "Product deleted successfully")
}

// Upload product image godoc
//
//	@Summary		Upload image for a shop product
//	@Tags			shop
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Product ID"
//	@Param			file	formData	file	true	"Product image (JPEG, PNG or WebP)"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop/{id}/image [post]
func (h *productHandler) UploadProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.UploadProductImage(c.Request.Context(), productID, file, authCtx)
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
