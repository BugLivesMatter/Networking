package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
}

// userRepositoryImpl реализует интерфейс UserRepository
type userRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository создаёт новый экземпляр репозитория
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

// Create создаёт нового пользователя в БД
func (r *userRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID находит пользователя по ID
func (r *userRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail находит пользователя по email
func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByYandexID находит пользователя по Yandex ID
func (r *userRepositoryImpl) GetByYandexID(ctx context.Context, yandexID string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("yandex_id = ?", yandexID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByVKID находит пользователя по VK ID
func (r *userRepositoryImpl) GetByVKID(ctx context.Context, vkID string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("vk_id = ?", vkID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update обновляет данные пользователя
func (r *userRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete мягко удаляет пользователя (soft delete)
func (r *userRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}
