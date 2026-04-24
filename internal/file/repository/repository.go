package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lab2/rest-api/internal/file/domain"
)

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByIDAndUserID(ctx context.Context, fileID, userID uuid.UUID) (*domain.File, error)
	GetByID(ctx context.Context, fileID uuid.UUID) (*domain.File, error)
	SoftDelete(ctx context.Context, fileID, userID uuid.UUID) error
}

type fileRepository struct {
	col *mongo.Collection
}

func NewFileRepository(col *mongo.Collection) FileRepository {
	return &fileRepository{col: col}
}

func (r *fileRepository) Create(ctx context.Context, file *domain.File) error {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	now := time.Now()
	file.CreatedAt = now
	file.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, file)
	return err
}

func (r *fileRepository) GetByIDAndUserID(ctx context.Context, fileID, userID uuid.UUID) (*domain.File, error) {
	return r.findOne(ctx, bson.M{"_id": fileID, "user_id": userID, "deleted_at": nil})
}

func (r *fileRepository) GetByID(ctx context.Context, fileID uuid.UUID) (*domain.File, error) {
	return r.findOne(ctx, bson.M{"_id": fileID, "deleted_at": nil})
}

func (r *fileRepository) SoftDelete(ctx context.Context, fileID, userID uuid.UUID) error {
	now := time.Now()
	filter := bson.M{"_id": fileID, "user_id": userID, "deleted_at": nil}
	update := bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}}
	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("файл не найден или уже удалён")
	}
	return nil
}

func (r *fileRepository) findOne(ctx context.Context, filter bson.M) (*domain.File, error) {
	var file domain.File
	if err := r.col.FindOne(ctx, filter).Decode(&file); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}
