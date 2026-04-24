package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lab2/rest-api/internal/auth/domain"
)

// TokenRepository определяет интерфейс для работы с refresh-токенами в БД
type TokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*domain.RefreshToken, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type tokenRepositoryImpl struct {
	col *mongo.Collection
}

// NewTokenRepository создаёт новый экземпляр репозитория
func NewTokenRepository(col *mongo.Collection) TokenRepository {
	return &tokenRepositoryImpl{col: col}
}

func (r *tokenRepositoryImpl) Create(ctx context.Context, token *domain.RefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	token.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, token)
	return err
}

func (r *tokenRepositoryImpl) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	return r.findOne(ctx, bson.M{"token_hash": tokenHash})
}

func (r *tokenRepositoryImpl) GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*domain.RefreshToken, error) {
	filter := bson.M{
		"access_token_hash": accessTokenHash,
		"revoked":           false,
		"expires_at":        bson.M{"$gt": time.Now()},
	}
	return r.findOne(ctx, filter)
}

func (r *tokenRepositoryImpl) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	filter := bson.M{
		"user_id":    userID,
		"revoked":    false,
		"expires_at": bson.M{"$gt": time.Now()},
	}
	cur, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var tokens []*domain.RefreshToken
	if err := cur.All(ctx, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *tokenRepositoryImpl) Revoke(ctx context.Context, tokenHash string) error {
	filter := bson.M{"token_hash": tokenHash}
	update := bson.M{"$set": bson.M{"revoked": true}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *tokenRepositoryImpl) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	filter := bson.M{"user_id": userID, "revoked": false}
	update := bson.M{"$set": bson.M{"revoked": true}}
	_, err := r.col.UpdateMany(ctx, filter, update)
	return err
}

func (r *tokenRepositoryImpl) DeleteExpired(ctx context.Context) error {
	filter := bson.M{"expires_at": bson.M{"$lt": time.Now()}}
	_, err := r.col.DeleteMany(ctx, filter)
	return err
}

func (r *tokenRepositoryImpl) findOne(ctx context.Context, filter bson.M) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	if err := r.col.FindOne(ctx, filter).Decode(&token); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}
