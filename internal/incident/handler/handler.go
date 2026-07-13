package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authdomain "github.com/lab2/rest-api/internal/auth/domain"
	authrepository "github.com/lab2/rest-api/internal/auth/repository"
	incidentdomain "github.com/lab2/rest-api/internal/incident/domain"
	incidentservice "github.com/lab2/rest-api/internal/incident/service"
	"github.com/lab2/rest-api/internal/storage"
)

type Handler struct {
	service *incidentservice.Service
	users   UserAdminRepository
	storage storage.Service
}

type UserAdminRepository interface {
	authrepository.UserRepository
	List(ctx context.Context) ([]*authdomain.User, error)
	UpdateRole(ctx context.Context, userID uuid.UUID, role authdomain.Role) (*authdomain.User, error)
}

func New(service *incidentservice.Service, users UserAdminRepository, storageService storage.Service) *Handler {
	return &Handler{service: service, users: users, storage: storageService}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	incidents := router.Group("/api/v1/incidents")
	incidents.GET("", h.List)
	incidents.POST("", h.Create)
	incidents.GET("/events", h.Events)
	incidents.GET("/:incidentID", h.Get)
	incidents.PATCH("/:incidentID", h.Patch)
	incidents.GET("/:incidentID/timeline", h.Timeline)
	incidents.POST("/:incidentID/comments", h.Comment)
	incidents.POST("/:incidentID/attachments", h.UploadAttachment)
	incidents.GET("/:incidentID/attachments/:fileID", h.DownloadAttachment)

	users := router.Group("/api/v1/users")
	users.GET("", h.ListUsers)
	users.PATCH("/:userID/role", h.UpdateRole)
}

func (h *Handler) List(c *gin.Context) {
	filter := incidentdomain.Filter{Page: intQuery(c, "page", 1), Limit: intQuery(c, "limit", 20), Service: c.Query("service")}
	if value := c.Query("status"); value != "" {
		filter.Status = incidentdomain.Status(value)
		if !filter.Status.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
	}
	if value := c.Query("severity"); value != "" {
		filter.Severity = incidentdomain.Severity(value)
		if !filter.Severity.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity"})
			return
		}
	}
	if value := c.Query("assigneeId"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assigneeId"})
			return
		}
		filter.AssigneeID = &id
	}
	page, err := h.service.List(c.Request.Context(), roleFrom(c), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": page.Data, "meta": gin.H{"page": page.Page, "limit": page.Limit, "total": page.Total, "totalPages": page.TotalPages}})
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	incident, err := h.service.Get(c.Request.Context(), roleFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, incident)
}

func (h *Handler) Create(c *gin.Context) {
	var input incidentservice.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	incident, err := h.service.Create(c.Request.Context(), userIDFrom(c), roleFrom(c), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, incident)
}

func (h *Handler) Patch(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var input incidentservice.PatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	incident, err := h.service.Patch(c.Request.Context(), userIDFrom(c), roleFrom(c), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, incident)
}

func (h *Handler) Timeline(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	events, err := h.service.Timeline(c.Request.Context(), roleFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) Comment(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var input struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	event, err := h.service.Comment(c.Request.Context(), userIDFrom(c), roleFrom(c), id, input.Message)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, event)
}

func (h *Handler) UploadAttachment(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	stream, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open attachment"})
		return
	}
	defer stream.Close()
	sniff := make([]byte, 512)
	n, _ := io.ReadFull(stream, sniff)
	mimetype := http.DetectContentType(sniff[:n])
	fullStream := io.MultiReader(bytes.NewReader(sniff[:n]), stream)
	file, err := h.service.UploadAttachment(c.Request.Context(), userIDFrom(c), roleFrom(c), id, fullStream, header.Size, header.Filename, mimetype)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"file": file.ToResponse()})
}

func (h *Handler) DownloadAttachment(c *gin.Context) {
	incidentID, ok := incidentID(c)
	if !ok {
		return
	}
	fileID, err := uuid.Parse(c.Param("fileID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fileID"})
		return
	}
	file, err := h.service.Attachment(c.Request.Context(), roleFrom(c), incidentID, fileID)
	if err != nil {
		writeError(c, err)
		return
	}
	object, err := h.storage.GetFileStream(c.Request.Context(), file.ObjectKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment object not found"})
		return
	}
	defer object.Reader.Close()
	contentType := object.ContentType
	if contentType == "" {
		contentType = file.Mimetype
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(file.OriginalName)))
	c.Header("Content-Length", strconv.FormatInt(object.Size, 10))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, object.Reader)
}

func (h *Handler) Events(c *gin.Context) {
	events, unsubscribe := h.service.Events()
	defer unsubscribe()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			_, _ = c.Writer.WriteString(": heartbeat\n\n")
			c.Writer.Flush()
		case event, open := <-events:
			if !open {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			c.SSEvent(event.Type, string(payload))
			c.Writer.Flush()
		}
	}
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.users.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list users"})
		return
	}
	responses := make([]*authdomain.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, user.ToResponse())
	}
	c.JSON(http.StatusOK, responses)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	if roleFrom(c) != authdomain.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}
	id, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userID"})
		return
	}
	var input struct {
		Role authdomain.Role `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !input.Role.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	user, err := h.users.UpdateRole(c.Request.Context(), id, input.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update role"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user.ToResponse())
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, incidentservice.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, incidentservice.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, incidentservice.ErrVersionConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, incidentservice.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func incidentID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("incidentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incidentID"})
		return uuid.Nil, false
	}
	return id, true
}

func userIDFrom(c *gin.Context) uuid.UUID {
	value, _ := c.Get("userID")
	id, _ := uuid.Parse(fmt.Sprint(value))
	return id
}

func roleFrom(c *gin.Context) authdomain.Role {
	value, _ := c.Get("userRole")
	role, _ := value.(authdomain.Role)
	return role
}

func intQuery(c *gin.Context, key string, fallback int64) int64 {
	value, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
