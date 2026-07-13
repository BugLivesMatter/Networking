package domain

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID           uuid.UUID  `bson:"_id"                  json:"id"`
	UserID       uuid.UUID  `bson:"user_id"              json:"userId"`
	OriginalName string     `bson:"original_name"        json:"originalName"`
	ObjectKey    string     `bson:"object_key"           json:"-"`
	Size         int64      `bson:"size"                 json:"size"`
	Mimetype     string     `bson:"mimetype"             json:"mimetype"`
	Bucket       string     `bson:"bucket"               json:"-"`
	Scope        string     `bson:"scope,omitempty"       json:"scope,omitempty"`
	IncidentID   *uuid.UUID `bson:"incident_id,omitempty" json:"incidentId,omitempty"`
	CreatedAt    time.Time  `bson:"created_at"           json:"createdAt"`
	UpdatedAt    time.Time  `bson:"updated_at"           json:"updatedAt"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty" json:"-"`
}

type FileResponse struct {
	ID           uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       uuid.UUID  `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	OriginalName string     `json:"originalName" example:"avatar.png"`
	Size         int64      `json:"size" example:"20480"`
	Mimetype     string     `json:"mimetype" example:"image/png"`
	Scope        string     `json:"scope,omitempty"`
	IncidentID   *uuid.UUID `json:"incidentId,omitempty"`
	CreatedAt    time.Time  `json:"createdAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
	UpdatedAt    time.Time  `json:"updatedAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
}

func (f *File) ToResponse() *FileResponse {
	return &FileResponse{
		ID:           f.ID,
		UserID:       f.UserID,
		OriginalName: f.OriginalName,
		Size:         f.Size,
		Mimetype:     f.Mimetype,
		Scope:        f.Scope,
		IncidentID:   f.IncidentID,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}
