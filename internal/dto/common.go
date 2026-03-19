package dto

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100
)

type Meta struct {
	Total      int64 `json:"total" example:"42"`
	Page       int   `json:"page" example:"1"`
	Limit      int   `json:"limit" example:"10"`
	TotalPages int   `json:"totalPages" example:"5"`
}
