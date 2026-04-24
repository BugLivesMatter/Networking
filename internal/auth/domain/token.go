package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken хранит информацию о сессиях пользователя.
// Оба токена хранятся в виде SHA-256 хэшей для безопасности.
type RefreshToken struct {
	ID              uuid.UUID `bson:"_id"                json:"id"`
	UserID          uuid.UUID `bson:"user_id"            json:"userId"`
	TokenHash       string    `bson:"token_hash"         json:"-"`
	AccessTokenHash string    `bson:"access_token_hash"  json:"-"`
	AccessJTI       string    `bson:"access_jti"         json:"-"`
	ExpiresAt       time.Time `bson:"expires_at"         json:"expiresAt"`
	Revoked         bool      `bson:"revoked"            json:"revoked"`
	CreatedAt       time.Time `bson:"created_at"         json:"createdAt"`
}

// IsExpired проверяет, истёк ли токен
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsActive возвращает true, если токен действителен и не отозван
func (t *RefreshToken) IsActive() bool {
	return !t.Revoked && !t.IsExpired()
}
