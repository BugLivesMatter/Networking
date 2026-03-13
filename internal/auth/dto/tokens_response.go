package dto

import "time"

// TokensResponse содержит пару токенов для клиента
type TokensResponse struct {
	AccessExpiresIn  time.Duration `json:"accessExpiresIn"`
	RefreshExpiresIn time.Duration `json:"refreshExpiresIn"`
}
