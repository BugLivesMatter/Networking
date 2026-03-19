package handler

// AppErrorResponse описывает единый формат ошибки для Swagger-аннотаций CRUD-эндпоинтов.
// Runtime ответы в коде приходят как JSON вида {"error": "..."}.
type AppErrorResponse struct {
	Error string `json:"error" example:"invalid category id"`
}

