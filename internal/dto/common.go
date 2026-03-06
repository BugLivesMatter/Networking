package dto

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100
)

type Meta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
}
