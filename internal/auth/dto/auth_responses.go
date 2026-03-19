package dto

import (
	"time"

	"github.com/google/uuid"
)

type RegisterResponse struct {
	Message string    `json:"message" example:"пользователь успешно зарегистрирован"`
	UserID  uuid.UUID `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type LoginResponse struct {
	Message          string `json:"message" example:"успешный вход"`
	AccessExpiresIn  string `json:"accessExpiresIn" example:"15m"`
	RefreshExpiresIn string `json:"refreshExpiresIn" example:"7d"`
}

type RefreshResponse struct {
	Message          string `json:"message" example:"токены обновлены"`
	AccessExpiresIn  string `json:"accessExpiresIn" example:"15m"`
	RefreshExpiresIn string `json:"refreshExpiresIn" example:"7d"`
}

type LogoutResponse struct {
	Message string `json:"message" example:"успешный выход"`
}

type LogoutAllResponse struct {
	Message string `json:"message" example:"успешный выход из всех сессий"`
}

type OAuthCallbackResponse struct {
	Message string    `json:"message" example:"успешный вход через OAuth"`
	UserID  uuid.UUID `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email   string    `json:"email" example:"student@example.com"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message" example:"если пользователь существует, письмо для сброса пароля отправлено"`
}

type ResetPasswordResponse struct {
	Message string `json:"message" example:"пароль успешно изменён"`
}

// ErrorResponse единый формат ошибки для Swagger-описаний auth-эндпоинтов.
// Runtime ответы в коде приходят как JSON с полем error.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid_or_expired_token"`
}

// Duration примечание: swagger плохо показывает time.Duration как пример,
// поэтому в ответах храним сроки как string.
var _ time.Duration

