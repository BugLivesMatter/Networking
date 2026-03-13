package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/auth/domain"
	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/repository"
)

// AuthService определяет интерфейс для бизнес-логики авторизации
type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*dto.TokensResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.TokensResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error)
}

// authServiceImpl реализует интерфейс AuthService
type authServiceImpl struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	passSvc   PasswordService
	jwtSvc    JWTService
}

// NewAuthService создаёт новый экземпляр сервиса авторизации
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	passSvc PasswordService,
	jwtSvc JWTService,
) AuthService {
	return &authServiceImpl{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		passSvc:   passSvc,
		jwtSvc:    jwtSvc,
	}
}

// Register регистрирует нового пользователя
func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*domain.User, error) {
	// Проверка валидности данных
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Проверка, что пользователь с таким email ещё не существует
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	// Хеширование пароля с уникальной солью
	passwordHash, salt, err := s.passSvc.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создание нового пользователя
	user := &domain.User{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		Salt:         salt,
	}

	// Сохранение в БД
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login выполняет вход пользователя и возвращает пару токенов
func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*dto.TokensResponse, error) {
	// Поиск пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("неверный email или пароль")
	}

	// Проверка пароля
	if err := s.passSvc.CheckPassword(password, user.PasswordHash, user.Salt); err != nil {
		return nil, errors.New("неверный email или пароль")
	}

	// Генерация JWT токенов
	accessToken, accessExpiry, err := s.jwtSvc.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiry, err := s.jwtSvc.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Хеширование refresh токена для хранения в БД
	refreshTokenHash := hashToken(refreshToken)

	// Сохранение refresh токена в БД
	token := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(refreshExpiry),
		Revoked:   false,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &dto.TokensResponse{
		AccessExpiresIn:  accessExpiry,
		RefreshExpiresIn: refreshExpiry,
	}, nil
}

// Refresh обновляет пару токенов по refresh токену
func (s *authServiceImpl) Refresh(ctx context.Context, refreshToken string) (*dto.TokensResponse, error) {
	// Валидация refresh токена
	claims, err := s.jwtSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("невалидный refresh токен")
	}

	// Хеширование токена для поиска в БД
	tokenHash := hashToken(refreshToken)

	// Поиск токена в БД
	storedToken, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if storedToken == nil {
		return nil, errors.New("токен не найден")
	}

	// Проверка, что токен не отозван и не истёк
	if !storedToken.IsActive() {
		return nil, errors.New("токен отозван или истёк")
	}

	// Отзыв старого токена
	if err := s.tokenRepo.Revoke(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Генерация новой пары токенов
	accessToken, accessExpiry, err := s.jwtSvc.GenerateAccessToken(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, refreshExpiry, err := s.jwtSvc.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохранение нового refresh токена
	newTokenHash := hashToken(newRefreshToken)
	newToken := &domain.RefreshToken{
		UserID:    claims.UserID,
		TokenHash: newTokenHash,
		ExpiresAt: time.Now().Add(refreshExpiry),
		Revoked:   false,
	}

	if err := s.tokenRepo.Create(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	return &dto.TokensResponse{
		AccessExpiresIn:  accessExpiry,
		RefreshExpiresIn: refreshExpiry,
	}, nil
}

// Logout завершает текущую сессию (отзывает один токен)
func (s *authServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.tokenRepo.Revoke(ctx, tokenHash)
}

// LogoutAll завершает все сессии пользователя
func (s *authServiceImpl) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.RevokeAll(ctx, userID)
}

// GetUserByID возвращает данные пользователя по ID (без чувствительных полей)
func (s *authServiceImpl) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("пользователь не найден")
	}

	return user.ToResponse(), nil
}

// hashToken хеширует токен для безопасного хранения в БД
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
