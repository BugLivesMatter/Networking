package dto

// ErrorResponse описывает единый формат ошибки для документации OpenAPI.
// В реальном API ответы часто приходят как {"error": "..."}; этот тип помогает
// swagger корректно отображать примеры полей.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid_or_expired_token"`
}

