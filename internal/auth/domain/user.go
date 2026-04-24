package domain

import (
	"time"

	"github.com/google/uuid"
)

// User представляет сущность пользователя в системе
type User struct {
	ID           uuid.UUID  `bson:"_id"                  json:"id"`
	Email        string     `bson:"email"                json:"email"`
	Phone        string     `bson:"phone,omitempty"      json:"phone,omitempty"`
	DisplayName  string     `bson:"display_name,omitempty" json:"displayName,omitempty"`
	Bio          string     `bson:"bio,omitempty"          json:"bio,omitempty"`
	AvatarFileID *uuid.UUID `bson:"avatar_file_id,omitempty" json:"avatarFileId,omitempty"`
	PasswordHash string     `bson:"password_hash"        json:"-"`
	Salt         string     `bson:"salt"                 json:"-"`
	YandexID     string     `bson:"yandex_id,omitempty"  json:"-"`
	VKID         string     `bson:"vk_id,omitempty"      json:"-"`
	CreatedAt    time.Time  `bson:"created_at"           json:"createdAt"`
	UpdatedAt    time.Time  `bson:"updated_at"           json:"updatedAt"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty" json:"-"`
}

// ToResponse возвращает безопасную версию пользователя для ответов API
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		DisplayName:  u.DisplayName,
		Bio:          u.Bio,
		AvatarFileID: u.AvatarFileID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// UserResponse — DTO для возврата данных пользователя клиенту
type UserResponse struct {
	ID           uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email        string     `json:"email" example:"student@example.com"`
	Phone        string     `json:"phone,omitempty" example:"+79991234567"`
	DisplayName  string     `json:"displayName,omitempty" example:"Иван Иванов"`
	Bio          string     `json:"bio,omitempty" example:"Backend разработчик"`
	AvatarFileID *uuid.UUID `json:"avatarFileId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedAt    time.Time  `json:"createdAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
	UpdatedAt    time.Time  `json:"updatedAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
}
