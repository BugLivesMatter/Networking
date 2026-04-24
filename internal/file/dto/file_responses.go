package dto

import "github.com/lab2/rest-api/internal/file/domain"

type UploadFileResponse struct {
	File *domain.FileResponse `json:"file"`
}
