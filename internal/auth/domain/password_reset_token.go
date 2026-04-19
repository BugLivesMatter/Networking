package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken представляет токен сброса пароля
type PasswordResetToken struct {
	ID        uuid.UUID `bson:"_id"        json:"id"`
	UserID    uuid.UUID `bson:"user_id"    json:"userId"`
	Token     string    `bson:"token"      json:"token"`
	ExpiresAt time.Time `bson:"expires_at" json:"expiresAt"`
	Used      bool      `bson:"used"       json:"used"`
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
}

// IsExpired проверяет, истёк ли токен
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
