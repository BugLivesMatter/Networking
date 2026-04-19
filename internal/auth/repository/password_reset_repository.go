package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lab2/rest-api/internal/auth/domain"
)

// PasswordResetRepository определяет интерфейс для работы с токенами сброса пароля
type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

type passwordResetRepositoryImpl struct {
	col *mongo.Collection
}

// NewPasswordResetRepository создаёт новый экземпляр репозитория
func NewPasswordResetRepository(col *mongo.Collection) PasswordResetRepository {
	return &passwordResetRepositoryImpl{col: col}
}

func (r *passwordResetRepositoryImpl) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	token.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, token)
	return err
}

func (r *passwordResetRepositoryImpl) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	filter := bson.M{
		"token":      token,
		"used":       false,
		"expires_at": bson.M{"$gt": time.Now()},
	}
	var resetToken domain.PasswordResetToken
	if err := r.col.FindOne(ctx, filter).Decode(&resetToken); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &resetToken, nil
}

func (r *passwordResetRepositoryImpl) MarkAsUsed(ctx context.Context, token string) error {
	filter := bson.M{"token": token}
	update := bson.M{"$set": bson.M{"used": true}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *passwordResetRepositoryImpl) DeleteExpired(ctx context.Context) error {
	filter := bson.M{"expires_at": bson.M{"$lt": time.Now()}}
	_, err := r.col.DeleteMany(ctx, filter)
	return err
}
