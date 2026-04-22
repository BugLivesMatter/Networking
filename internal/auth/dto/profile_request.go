package dto

import "github.com/google/uuid"

type UpdateProfileRequest struct {
	DisplayName  *string    `json:"displayName,omitempty" example:"Иван Иванов"`
	Bio          *string    `json:"bio,omitempty" example:"Backend разработчик"`
	AvatarFileID *uuid.UUID `json:"avatarFileId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
}
