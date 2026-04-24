package domain

import (
	"time"

	"github.com/google/uuid"
	categorydomain "github.com/lab2/rest-api/internal/category/domain"
)

type Product struct {
	ID          uuid.UUID                `bson:"_id"                  json:"id"`
	CategoryID  uuid.UUID                `bson:"category_id"          json:"categoryId"`
	Category    *categorydomain.Category `bson:"-"                    json:"category,omitempty"`
	Name        string                   `bson:"name"                 json:"name"`
	Description string                   `bson:"description"          json:"description"`
	Price       float64                  `bson:"price"                json:"price"`
	Status      string                   `bson:"status"               json:"status"`
	CreatedAt   time.Time                `bson:"created_at"           json:"createdAt"`
	UpdatedAt   time.Time                `bson:"updated_at"           json:"updatedAt"`
	DeletedAt   *time.Time               `bson:"deleted_at,omitempty" json:"-"`
}
