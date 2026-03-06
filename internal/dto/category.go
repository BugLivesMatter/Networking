package dto

import "github.com/lab2/rest-api/internal/domain"

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"omitempty,oneof=active hidden"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active hidden"`
}

type PatchCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status" binding:"omitempty,oneof=active hidden"`
}

type CategoryResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
}

type CategoryListResponse struct {
	Data []CategoryResponse `json:"data"`
	Meta Meta               `json:"meta"`
}

func CategoryToResponse(c *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID.String(),
		Name:        c.Name,
		Description: c.Description,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}
