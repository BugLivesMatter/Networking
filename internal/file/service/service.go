package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/cache"
	filedomain "github.com/lab2/rest-api/internal/file/domain"
	"github.com/lab2/rest-api/internal/file/repository"
	"github.com/lab2/rest-api/internal/storage"
)

var allowedAvatarMimeTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/jpg":  {},
}

type Service interface {
	Upload(ctx context.Context, userID uuid.UUID, stream io.Reader, size int64, filename, mimetype string) (*filedomain.File, error)
	GetByID(ctx context.Context, fileID, userID uuid.UUID) (*filedomain.File, error)
	List(ctx context.Context, userID uuid.UUID) ([]*filedomain.File, error)
	Delete(ctx context.Context, fileID, userID uuid.UUID) error
}

type fileService struct {
	repo        repository.FileRepository
	storage     storage.Service
	cacheSvc    cache.Service
	cacheTTL    time.Duration
	bucketName  string
	maxFileSize int64
}

func NewService(
	repo repository.FileRepository,
	storageSvc storage.Service,
	cacheSvc cache.Service,
	cacheTTL time.Duration,
	bucketName string,
	maxFileSize int64,
) Service {
	return &fileService{
		repo:        repo,
		storage:     storageSvc,
		cacheSvc:    cacheSvc,
		cacheTTL:    cacheTTL,
		bucketName:  bucketName,
		maxFileSize: maxFileSize,
	}
}

func (s *fileService) Upload(ctx context.Context, userID uuid.UUID, stream io.Reader, size int64, filename, mimetype string) (*filedomain.File, error) {
	if size <= 0 {
		return nil, errors.New("файл пустой")
	}
	if size > s.maxFileSize {
		return nil, fmt.Errorf("размер файла превышает лимит %d байт", s.maxFileSize)
	}
	if _, ok := allowedAvatarMimeTypes[strings.ToLower(mimetype)]; !ok {
		return nil, errors.New("неподдерживаемый MIME-тип файла")
	}

	objectKey, err := s.storage.UploadFile(ctx, stream, size, filename, mimetype, userID)
	if err != nil {
		return nil, err
	}

	file := &filedomain.File{
		UserID:       userID,
		OriginalName: filename,
		ObjectKey:    objectKey,
		Size:         size,
		Mimetype:     mimetype,
		Bucket:       s.bucketName,
	}
	if err := s.repo.Create(ctx, file); err != nil {
		if cleanupErr := s.storage.DeleteFile(ctx, objectKey); cleanupErr != nil {
			log.Printf("WARN: не удалось удалить осиротевший объект MinIO %s: %v", objectKey, cleanupErr)
		}
		return nil, fmt.Errorf("сохранение метаданных файла: %w", err)
	}

	return file, nil
}

func (s *fileService) GetByID(ctx context.Context, fileID, userID uuid.UUID) (*filedomain.File, error) {
	cacheKey := cache.FileMetaKey(fileID)
	var cached filedomain.File
	hit, err := s.cacheSvc.Get(ctx, cacheKey, &cached)
	if err == nil && hit {
		if cached.UserID != userID || cached.DeletedAt != nil {
			return nil, errors.New("файл не найден")
		}
		return &cached, nil
	}

	file, err := s.repo.GetByIDAndUserID(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, errors.New("файл не найден")
	}
	_ = s.cacheSvc.Set(ctx, cacheKey, file, s.cacheTTL)
	return file, nil
}

func (s *fileService) List(ctx context.Context, userID uuid.UUID) ([]*filedomain.File, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *fileService) Delete(ctx context.Context, fileID, userID uuid.UUID) error {
	file, err := s.repo.GetByIDAndUserID(ctx, fileID, userID)
	if err != nil {
		return err
	}
	if file == nil {
		return errors.New("файл не найден")
	}

	if err := s.repo.SoftDelete(ctx, fileID, userID); err != nil {
		return err
	}
	_ = s.cacheSvc.Del(ctx, cache.FileMetaKey(fileID))
	_ = s.storage.DeleteFile(ctx, file.ObjectKey)
	return nil
}

