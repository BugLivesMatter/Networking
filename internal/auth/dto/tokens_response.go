package dto

import "time"

// TokensResponse содержит пару токенов для клиента
type TokensResponse struct {
	AccessToken      string        `json:"-"` // Не показывать в JSON ответе
	RefreshToken     string        `json:"-"` // Не показывать в JSON ответе
	AccessExpiresIn  time.Duration `json:"accessExpiresIn"`
	RefreshExpiresIn time.Duration `json:"refreshExpiresIn"`
}
