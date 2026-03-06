package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lab2/rest-api/internal/dto"
	"github.com/lab2/rest-api/internal/service"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	c.JSON(http.StatusCreated, dto.ProductToResponse(product))
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	product, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

func (h *ProductHandler) List(c *gin.Context) {
	page := 1
	limit := dto.DefaultLimit
	var categoryID *uuid.UUID
	if cid := c.Query("category_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			categoryID = &id
		}
	}
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v >= 1 && v <= dto.MaxLimit {
			limit = v
		}
	}
	products, total, totalPages, err := h.svc.List(c.Request.Context(), page, limit, categoryID)
	if err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	data := make([]dto.ProductResponse, len(products))
	for i := range products {
		data[i] = dto.ProductToResponse(&products[i])
	}
	c.JSON(http.StatusOK, dto.ProductListResponse{
		Data: data,
		Meta: dto.Meta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

func (h *ProductHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

func (h *ProductHandler) Patch(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.PatchProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Patch(c.Request.Context(), id, &req)
	if err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

func (h *ProductHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		status := statusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err, status)})
		return
	}
	c.Status(http.StatusNoContent)
}
