package dto

import "github.com/lab2/rest-api/internal/domain"

type CreateProductRequest struct {
	CategoryID  string  `json:"categoryId" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gte=0"`
	Status      string  `json:"status" binding:"omitempty,oneof=available out_of_stock discontinued"`
}

type UpdateProductRequest struct {
	CategoryID  string  `json:"categoryId" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gte=0"`
	Status      string  `json:"status" binding:"required,oneof=available out_of_stock discontinued"`
}

type PatchProductRequest struct {
	CategoryID  *string  `json:"categoryId"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price" binding:"omitempty,gte=0"`
	Status      *string  `json:"status" binding:"omitempty,oneof=available out_of_stock discontinued"`
}

type ProductResponse struct {
	ID           string  `json:"id"`
	CategoryID   string  `json:"categoryId"`
	CategoryName string  `json:"categoryName,omitempty"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"createdAt"`
}

type ProductListResponse struct {
	Data []ProductResponse `json:"data"`
	Meta Meta              `json:"meta"`
}

func ProductToResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID.String(),
		CategoryID:  p.CategoryID.String(),
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	if p.Category != nil {
		resp.CategoryName = p.Category.Name
	}
	return resp
}
