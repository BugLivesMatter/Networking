package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID  `bson:"_id"                  json:"id"`
	Name        string     `bson:"name"                 json:"name"`
	Description string     `bson:"description"          json:"description"`
	Status      string     `bson:"status"               json:"status"`
	CreatedAt   time.Time  `bson:"created_at"           json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updated_at"           json:"updatedAt"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty" json:"-"`
}
