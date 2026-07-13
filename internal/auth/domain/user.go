package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleViewer          Role = "viewer"
	RoleResponder       Role = "responder"
	RoleIncidentManager Role = "incident-manager"
	RoleAdmin           Role = "admin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleViewer, RoleResponder, RoleIncidentManager, RoleAdmin:
		return true
	default:
		return false
	}
}

func ParseRole(value string) (Role, error) {
	role := Role(value)
	if !role.Valid() {
		return "", fmt.Errorf("invalid role %q", value)
	}
	return role, nil
}

func (r Role) AtLeast(required Role) bool {
	rank := map[Role]int{RoleViewer: 0, RoleResponder: 1, RoleIncidentManager: 2, RoleAdmin: 3}
	return rank[r] >= rank[required]
}

// User представляет сущность пользователя в системе
type User struct {
	ID           uuid.UUID  `bson:"_id"                  json:"id"`
	Email        string     `bson:"email"                json:"email"`
	Phone        string     `bson:"phone,omitempty"      json:"phone,omitempty"`
	DisplayName  string     `bson:"display_name,omitempty" json:"displayName,omitempty"`
	Bio          string     `bson:"bio,omitempty"          json:"bio,omitempty"`
	Role         Role       `bson:"role"                    json:"role"`
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
	role := u.Role
	if !role.Valid() {
		role = RoleViewer
	}
	return &UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		DisplayName:  u.DisplayName,
		Bio:          u.Bio,
		Role:         role,
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
	Role         Role       `json:"role" example:"viewer"`
	AvatarFileID *uuid.UUID `json:"avatarFileId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedAt    time.Time  `json:"createdAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
	UpdatedAt    time.Time  `json:"updatedAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
}
