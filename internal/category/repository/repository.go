package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lab2/rest-api/internal/category/domain"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	List(ctx context.Context, offset, limit int) ([]domain.Category, int64, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type categoryRepository struct {
	col *mongo.Collection
}

func NewCategoryRepository(col *mongo.Collection) CategoryRepository {
	return &categoryRepository{col: col}
}

func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) error {
	if category.ID == uuid.Nil {
		category.ID = uuid.New()
	}
	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now
	if category.Status == "" {
		category.Status = "active"
	}
	_, err := r.col.InsertOne(ctx, category)
	return err
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	filter := bson.M{"_id": id, "deleted_at": nil}
	var category domain.Category
	err := r.col.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) List(ctx context.Context, offset, limit int) ([]domain.Category, int64, error) {
	filter := bson.M{"deleted_at": nil}
	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().SetSkip(int64(offset)).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var categories []domain.Category
	if err := cur.All(ctx, &categories); err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *domain.Category) error {
	category.UpdatedAt = time.Now()
	filter := bson.M{"_id": category.ID}
	update := bson.M{"$set": category}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}
