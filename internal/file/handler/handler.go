package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/file/dto"
	"github.com/lab2/rest-api/internal/file/service"
	"github.com/lab2/rest-api/internal/storage"
)

type Handler struct {
	fileService    service.Service
	storageService storage.Service
}

func NewHandler(fileService service.Service, storageService storage.Service) *Handler {
	return &Handler{
		fileService:    fileService,
		storageService: storageService,
	}
}

// List godoc
// @Summary Список файлов текущего пользователя
// @Tags files
// @Produce json
// @Security CookieAuth
// @Success 200 {array} domain.FileResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /files [get]
func (h *Handler) List(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	files, err := h.fileService.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения списка файлов"})
		return
	}
	resp := make([]interface{}, len(files))
	for i, f := range files {
		resp[i] = f.ToResponse()
	}
	c.JSON(http.StatusOK, resp)
}

// Upload godoc
// @Summary Загрузка файла
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Security CookieAuth
// @Param file formData file true "Файл"
// @Success 201 {object} dto.UploadFileResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /files [post]
func (h *Handler) Upload(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл обязателен"})
		return
	}
	fileStream, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось открыть файл"})
		return
	}
	defer fileStream.Close()

	// Detect MIME from file bytes to prevent Content-Type header spoofing
	sniff := make([]byte, 512)
	n, _ := io.ReadFull(fileStream, sniff)
	detectedMime := http.DetectContentType(sniff[:n])
	fullStream := io.MultiReader(bytes.NewReader(sniff[:n]), fileStream)

	file, err := h.fileService.Upload(
		c.Request.Context(),
		userID,
		fullStream,
		fileHeader.Size,
		fileHeader.Filename,
		detectedMime,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.UploadFileResponse{File: file.ToResponse()})
}

// Download godoc
// @Summary Скачивание файла по ID
// @Tags files
// @Produce octet-stream
// @Security CookieAuth
// @Param fileId path string true "ID файла"
// @Success 200 {file} binary
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /files/{fileId} [get]
func (h *Handler) Download(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный fileId"})
		return
	}

	file, err := h.fileService.GetByID(c.Request.Context(), fileID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "файл не найден"})
		return
	}

	streamObj, err := h.storageService.GetFileStream(c.Request.Context(), file.ObjectKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "объект не найден в хранилище"})
		return
	}
	defer streamObj.Reader.Close()

	contentType := streamObj.ContentType
	if contentType == "" {
		contentType = file.Mimetype
	}
	c.Header("Content-Type", contentType)
	// RFC 5987 encoding for Unicode filenames
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(file.OriginalName)))
	c.Header("Content-Length", fmt.Sprintf("%d", streamObj.Size))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, streamObj.Reader)
}

// Delete godoc
// @Summary Удаление файла
// @Tags files
// @Security CookieAuth
// @Param fileId path string true "ID файла"
// @Success 204
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /files/{fileId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный fileId"})
		return
	}

	if err := h.fileService.Delete(c.Request.Context(), fileID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func userIDFromContext(c *gin.Context) (uuid.UUID, error) {
	rawUserID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, fmt.Errorf("пользователь не авторизован")
	}
	userID, err := uuid.Parse(rawUserID.(string))
	if err != nil {
		return uuid.Nil, fmt.Errorf("ошибка идентификатора пользователя")
	}
	return userID, nil
}
