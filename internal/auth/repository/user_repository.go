package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lab2/rest-api/internal/auth/domain"
)

// UserRepository определяет интерфейс для работы с пользователями в БД
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByYandexID(ctx context.Context, yandexID string) (*domain.User, error)
	GetByVKID(ctx context.Context, vkID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash, salt string) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, displayName, bio *string, avatarFileID *uuid.UUID) error
}

type userRepositoryImpl struct {
	col *mongo.Collection
}

// NewUserRepository создаёт новый экземпляр репозитория
func NewUserRepository(col *mongo.Collection) *userRepositoryImpl {
	return &userRepositoryImpl{col: col}
}

func (r *userRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	if !user.Role.Valid() {
		var state struct {
			Sequence int64 `bson:"sequence"`
		}
		err := r.col.Database().Collection("bootstrap_state").FindOneAndUpdate(
			ctx,
			bson.M{"_id": "users"},
			bson.M{"$inc": bson.M{"sequence": 1}, "$set": bson.M{"updated_at": now}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		).Decode(&state)
		if err != nil {
			return err
		}
		if state.Sequence == 1 {
			user.Role = domain.RoleAdmin
		} else {
			user.Role = domain.RoleViewer
		}
	}
	_, err := r.col.InsertOne(ctx, user)
	return err
}

func (r *userRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"_id": id, "deleted_at": nil})
}

func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"email": email, "deleted_at": nil})
}

func (r *userRepositoryImpl) GetByYandexID(ctx context.Context, yandexID string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"yandex_id": yandexID, "deleted_at": nil})
}

func (r *userRepositoryImpl) GetByVKID(ctx context.Context, vkID string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"vk_id": vkID, "deleted_at": nil})
}

func (r *userRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	filter := bson.M{"_id": user.ID, "deleted_at": nil}
	// Explicit field map prevents accidental _id update (MongoDB error) and zero-value overwrites
	update := bson.M{"$set": bson.M{
		"email":          user.Email,
		"phone":          user.Phone,
		"display_name":   user.DisplayName,
		"bio":            user.Bio,
		"role":           user.Role,
		"avatar_file_id": user.AvatarFileID,
		"yandex_id":      user.YandexID,
		"vk_id":          user.VKID,
		"updated_at":     user.UpdatedAt,
	}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepositoryImpl) List(ctx context.Context) ([]*domain.User, error) {
	cursor, err := r.col.Find(ctx, bson.M{"deleted_at": nil}, options.Find().SetSort(bson.D{{Key: "display_name", Value: 1}, {Key: "email", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []*domain.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepositoryImpl) UpdateRole(ctx context.Context, userID uuid.UUID, role domain.Role) (*domain.User, error) {
	if !role.Valid() {
		return nil, errors.New("invalid role")
	}
	var user domain.User
	err := r.col.FindOneAndUpdate(
		ctx,
		bson.M{"_id": userID, "deleted_at": nil},
		bson.M{"$set": bson.M{"role": role, "updated_at": time.Now()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	filter := bson.M{"_id": id, "deleted_at": nil}
	update := bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepositoryImpl) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash, salt string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$set": bson.M{
		"password_hash": passwordHash,
		"salt":          salt,
		"updated_at":    time.Now(),
	}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepositoryImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName, bio *string, avatarFileID *uuid.UUID) error {
	filter := bson.M{"_id": userID, "deleted_at": nil}
	setFields := bson.M{
		"updated_at": time.Now(),
	}
	if displayName != nil {
		setFields["display_name"] = *displayName
	}
	if bio != nil {
		setFields["bio"] = *bio
	}
	if avatarFileID != nil {
		setFields["avatar_file_id"] = *avatarFileID
	}

	update := bson.M{"$set": setFields}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepositoryImpl) findOne(ctx context.Context, filter bson.M) (*domain.User, error) {
	var user domain.User
	if err := r.col.FindOne(ctx, filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
