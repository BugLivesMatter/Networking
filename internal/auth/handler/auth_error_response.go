package handler

// AuthErrorResponse описывает единый формат ошибки для Swagger-аннотаций auth-эндпоинтов.
// Runtime ответы в коде приходят как JSON вида {"error": "..."}.
type AuthErrorResponse struct {
	Error string `json:"error" example:"invalid_or_expired_token"`
}

