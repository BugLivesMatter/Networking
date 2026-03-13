package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/lab2/rest-api/internal/auth/domain"
)

// PasswordResetRepository определяет интерфейс для работы с токенами сброса пароля
type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// passwordResetRepositoryImpl реализует интерфейс PasswordResetRepository
type passwordResetRepositoryImpl struct {
	db *gorm.DB
}

// NewPasswordResetRepository создаёт новый экземпляр репозитория
func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepositoryImpl{db: db}
}

// Create создаёт новый токен сброса пароля
func (r *passwordResetRepositoryImpl) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetByToken находит токен по значению
func (r *passwordResetRepositoryImpl) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	var resetToken domain.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&resetToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &resetToken, nil
}

// MarkAsUsed помечает токен как использованный
func (r *passwordResetRepositoryImpl) MarkAsUsed(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Model(&domain.PasswordResetToken{}).
		Where("token = ?", token).
		Update("used", true).Error
}

// DeleteExpired удаляет истёкшие токены
func (r *passwordResetRepositoryImpl) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&domain.PasswordResetToken{}).Error
}
