package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	categorydomain "github.com/lab2/rest-api/internal/category/domain"
	"github.com/lab2/rest-api/internal/product/domain"
)

type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, offset, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByCategoryID(ctx context.Context, categoryID uuid.UUID) (int64, error)
}

type productRepository struct {
	col     *mongo.Collection
	catCol  *mongo.Collection
}

func NewProductRepository(col, catCol *mongo.Collection) ProductRepository {
	return &productRepository{col: col, catCol: catCol}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	if product.Status == "" {
		product.Status = "available"
	}
	_, err := r.col.InsertOne(ctx, product)
	return err
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	filter := bson.M{"_id": id, "deleted_at": nil}
	var product domain.Product
	if err := r.col.FindOne(ctx, filter).Decode(&product); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	r.fillCategory(ctx, &product)
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, offset, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, error) {
	filter := bson.M{"deleted_at": nil}
	if categoryID != nil {
		filter["category_id"] = *categoryID
	}
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
	var products []domain.Product
	if err := cur.All(ctx, &products); err != nil {
		return nil, 0, err
	}
	for i := range products {
		r.fillCategory(ctx, &products[i])
	}
	return products, total, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	product.UpdatedAt = time.Now()
	filter := bson.M{"_id": product.ID}
	update := bson.M{"$set": product}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"deleted_at": now}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *productRepository) CountByCategoryID(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	filter := bson.M{"category_id": categoryID, "deleted_at": nil}
	return r.col.CountDocuments(ctx, filter)
}

// fillCategory заполняет поле Category из коллекции categories (замена GORM Preload).
func (r *productRepository) fillCategory(ctx context.Context, product *domain.Product) {
	var cat categorydomain.Category
	if err := r.catCol.FindOne(ctx, bson.M{"_id": product.CategoryID}).Decode(&cat); err == nil {
		product.Category = &cat
	}
}
