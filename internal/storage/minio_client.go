package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/lab2/rest-api/internal/config"
)

type FileObject struct {
	Reader       io.ReadCloser
	Size         int64
	ContentType  string
	OriginalName string
}

type Service interface {
	UploadFile(ctx context.Context, reader io.Reader, size int64, filename, mimetype string, userID uuid.UUID) (string, error)
	GetFileStream(ctx context.Context, objectKey string) (*FileObject, error)
	DeleteFile(ctx context.Context, objectKey string) error
	FileExists(ctx context.Context, objectKey string) (bool, error)
}

type minioService struct {
	client *minio.Client
	bucket string
}

func NewMinIOService(cfg *config.Config) (Service, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("инициализация minio клиента: %w", err)
	}

	svc := &minioService{
		client: client,
		bucket: cfg.MinIOBucket,
	}
	if err := svc.ensureBucket(context.Background()); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *minioService) UploadFile(ctx context.Context, reader io.Reader, size int64, filename, mimetype string, userID uuid.UUID) (string, error) {
	objectKey := buildObjectKey(userID, filename)
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: mimetype,
	})
	if err != nil {
		return "", fmt.Errorf("загрузка файла в minio: %w", err)
	}
	return objectKey, nil
}

func (s *minioService) GetFileStream(ctx context.Context, objectKey string) (*FileObject, error) {
	object, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("получение объекта из minio: %w", err)
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("получение метаданных объекта из minio: %w", err)
	}

	return &FileObject{
		Reader:      object,
		Size:        info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *minioService) DeleteFile(ctx context.Context, objectKey string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("удаление объекта из minio: %w", err)
	}
	return nil
}

func (s *minioService) FileExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject" {
		return false, nil
	}
	return false, fmt.Errorf("проверка существования объекта в minio: %w", err)
}

func (s *minioService) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("проверка существования bucket в minio: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("создание bucket в minio: %w", err)
	}
	return nil
}

func buildObjectKey(userID uuid.UUID, filename string) string {
	ext := ""
	if dot := strings.LastIndex(filename, "."); dot >= 0 {
		ext = filename[dot:]
	}
	return fmt.Sprintf("users/%s/%s%s", userID.String(), uuid.NewString(), ext)
}
