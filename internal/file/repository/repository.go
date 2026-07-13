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
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.File, error)
	SoftDelete(ctx context.Context, fileID, userID uuid.UUID) error
}

func (r *fileRepository) GetIncidentAttachment(ctx context.Context, fileID, incidentID uuid.UUID) (*domain.File, error) {
	return r.findOne(ctx, bson.M{"_id": fileID, "scope": "incident", "incident_id": incidentID, "deleted_at": nil})
}

type fileRepository struct {
	col *mongo.Collection
}

func NewFileRepository(col *mongo.Collection) *fileRepository {
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
	return r.findOne(ctx, legacyFileFilter(bson.M{"_id": fileID, "user_id": userID, "deleted_at": nil}))
}

func (r *fileRepository) GetByID(ctx context.Context, fileID uuid.UUID) (*domain.File, error) {
	return r.findOne(ctx, legacyFileFilter(bson.M{"_id": fileID, "deleted_at": nil}))
}

func (r *fileRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.File, error) {
	cursor, err := r.col.Find(ctx, legacyFileFilter(bson.M{"user_id": userID, "deleted_at": nil}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var files []*domain.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *fileRepository) SoftDelete(ctx context.Context, fileID, userID uuid.UUID) error {
	now := time.Now()
	filter := legacyFileFilter(bson.M{"_id": fileID, "user_id": userID, "deleted_at": nil})
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

// Incident attachments have a separate access path and must not accidentally
// appear in the legacy avatar/files API.
func legacyFileFilter(filter bson.M) bson.M {
	filter["$or"] = bson.A{bson.M{"scope": bson.M{"$exists": false}}, bson.M{"scope": bson.M{"$ne": "incident"}}}
	return filter
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
