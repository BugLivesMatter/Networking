package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	authdomain "github.com/lab2/rest-api/internal/auth/domain"
	clusterdomain "github.com/lab2/rest-api/internal/cluster/domain"
	clustersource "github.com/lab2/rest-api/internal/cluster/source"
	filedomain "github.com/lab2/rest-api/internal/file/domain"
	incidentdomain "github.com/lab2/rest-api/internal/incident/domain"
	"github.com/lab2/rest-api/internal/incident/hub"
	incidentrepository "github.com/lab2/rest-api/internal/incident/repository"
	"github.com/lab2/rest-api/internal/storage"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("incident not found")
	ErrValidation      = errors.New("validation error")
	ErrVersionConflict = incidentrepository.ErrVersionConflict
)

type CreateInput struct {
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	Service       string                  `json:"service"`
	Severity      incidentdomain.Severity `json:"severity"`
	AssigneeID    *uuid.UUID              `json:"assigneeId"`
	SourceEventID string                  `json:"sourceEventId"`
}

type PatchInput struct {
	Version     int64                    `json:"version"`
	Title       *string                  `json:"title"`
	Description *string                  `json:"description"`
	Service     *string                  `json:"service"`
	Severity    *incidentdomain.Severity `json:"severity"`
	Status      *incidentdomain.Status   `json:"status"`
	AssigneeID  *uuid.UUID               `json:"assigneeId"`
	Unassign    bool                     `json:"unassign"`
}

type Service struct {
	repo        incidentrepository.Repository
	cluster     clustersource.ClusterSource
	hub         *hub.Hub
	files       AttachmentRepository
	storage     storage.Service
	bucket      string
	maxFileSize int64
	users       UserLookup
}

// AttachmentRepository is intentionally narrower than the legacy file
// repository interface, keeping existing avatar/file test doubles compatible.
type AttachmentRepository interface {
	Create(ctx context.Context, file *filedomain.File) error
	GetIncidentAttachment(ctx context.Context, fileID, incidentID uuid.UUID) (*filedomain.File, error)
}

type UserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*authdomain.User, error)
}

func New(repo incidentrepository.Repository, cluster clustersource.ClusterSource, eventHub *hub.Hub, files AttachmentRepository, storageService storage.Service, bucket string, maxFileSize int64, users ...UserLookup) *Service {
	if eventHub == nil {
		eventHub = hub.New()
	}
	service := &Service{repo: repo, cluster: cluster, hub: eventHub, files: files, storage: storageService, bucket: bucket, maxFileSize: maxFileSize}
	if len(users) > 0 {
		service.users = users[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, role authdomain.Role, filter incidentdomain.Filter) (*incidentdomain.Page, error) {
	if !role.Valid() {
		return nil, ErrForbidden
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) Get(ctx context.Context, role authdomain.Role, id uuid.UUID) (*incidentdomain.Incident, error) {
	if !role.Valid() {
		return nil, ErrForbidden
	}
	incident, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, ErrNotFound
	}
	return incident, nil
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, role authdomain.Role, input CreateInput) (*incidentdomain.Incident, error) {
	if !role.AtLeast(authdomain.RoleResponder) {
		return nil, ErrForbidden
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Service = strings.TrimSpace(input.Service)
	var sourceEvent *clusterdomain.Event
	if input.SourceEventID != "" {
		snapshot, err := s.cluster.Snapshot(ctx)
		if err != nil {
			return nil, fmt.Errorf("cluster snapshot: %w", err)
		}
		for _, event := range snapshot.Events {
			if event.ID == input.SourceEventID {
				copy := event
				sourceEvent = &copy
				break
			}
		}
		if sourceEvent == nil {
			return nil, fmt.Errorf("%w: source event not found", ErrValidation)
		}
		if input.Title == "" {
			input.Title = sourceEvent.Title
		}
		if input.Description == "" {
			input.Description = sourceEvent.Detail
		}
		if input.Service == "" {
			input.Service = sourceEvent.ServiceID
		}
		if !input.Severity.Valid() {
			input.Severity = severityFromHealth(sourceEvent.Status)
		}
	}
	if input.Title == "" || input.Service == "" {
		return nil, fmt.Errorf("%w: title and service are required", ErrValidation)
	}
	if !input.Severity.Valid() {
		input.Severity = incidentdomain.SeverityMedium
	}
	if input.AssigneeID != nil && !role.AtLeast(authdomain.RoleIncidentManager) && *input.AssigneeID != actor {
		return nil, ErrForbidden
	}
	if input.AssigneeID != nil {
		if err := s.validateAssignee(ctx, *input.AssigneeID); err != nil {
			return nil, err
		}
	}
	incident := &incidentdomain.Incident{
		Title: input.Title, Description: input.Description, Service: input.Service,
		Severity: input.Severity, Status: incidentdomain.StatusOpen, AssigneeID: input.AssigneeID,
		CreatorID: actor, SourceEvent: sourceEvent,
	}
	timeline := &incidentdomain.TimelineEvent{Type: incidentdomain.TimelineCreated, ActorID: actor, Message: "Incident created"}
	if err := s.repo.Create(ctx, incident, timeline); err != nil {
		return nil, err
	}
	s.publish("incident.created", incident.ID, incident)
	s.publish("timeline.appended", incident.ID, timeline)
	return incident, nil
}

func (s *Service) Patch(ctx context.Context, actor uuid.UUID, role authdomain.Role, id uuid.UUID, input PatchInput) (*incidentdomain.Incident, error) {
	if !role.AtLeast(authdomain.RoleResponder) {
		return nil, ErrForbidden
	}
	if input.Version < 1 {
		return nil, fmt.Errorf("%w: version is required", ErrValidation)
	}
	incident, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, ErrNotFound
	}
	if incident.Version != input.Version {
		return nil, ErrVersionConflict
	}
	manager := role.AtLeast(authdomain.RoleIncidentManager)
	changes := make([]incidentdomain.TimelineEvent, 0, 4)
	if input.Title != nil || input.Description != nil || input.Service != nil {
		if !manager {
			return nil, ErrForbidden
		}
		fields := map[string]interface{}{}
		if input.Title != nil {
			value := strings.TrimSpace(*input.Title)
			if value == "" {
				return nil, fmt.Errorf("%w: title cannot be empty", ErrValidation)
			}
			fields["title"] = map[string]interface{}{"from": incident.Title, "to": value}
			incident.Title = value
		}
		if input.Description != nil {
			fields["description"] = map[string]interface{}{"from": incident.Description, "to": *input.Description}
			incident.Description = strings.TrimSpace(*input.Description)
		}
		if input.Service != nil {
			value := strings.TrimSpace(*input.Service)
			if value == "" {
				return nil, fmt.Errorf("%w: service cannot be empty", ErrValidation)
			}
			fields["service"] = map[string]interface{}{"from": incident.Service, "to": value}
			incident.Service = value
		}
		changes = append(changes, incidentdomain.TimelineEvent{Type: incidentdomain.TimelineIncidentEdited, ActorID: actor, Changes: fields})
	}
	if input.Severity != nil && *input.Severity != incident.Severity {
		if !manager {
			return nil, ErrForbidden
		}
		if !input.Severity.Valid() {
			return nil, fmt.Errorf("%w: invalid severity", ErrValidation)
		}
		changes = append(changes, changeEvent(incidentdomain.TimelineSeverityChanged, actor, "severity", incident.Severity, *input.Severity))
		incident.Severity = *input.Severity
	}
	if input.Status != nil && *input.Status != incident.Status {
		if !input.Status.Valid() || !validTransition(incident.Status, *input.Status) {
			return nil, fmt.Errorf("%w: invalid status transition", ErrValidation)
		}
		if !manager && *input.Status == incidentdomain.StatusResolved {
			return nil, ErrForbidden
		}
		changes = append(changes, changeEvent(incidentdomain.TimelineStatusChanged, actor, "status", incident.Status, *input.Status))
		incident.Status = *input.Status
	}
	if input.AssigneeID != nil || input.Unassign {
		var next *uuid.UUID
		if !input.Unassign {
			next = input.AssigneeID
		}
		if !manager && (next == nil || *next != actor || (incident.AssigneeID != nil && *incident.AssigneeID != actor)) {
			return nil, ErrForbidden
		}
		if next != nil {
			if err := s.validateAssignee(ctx, *next); err != nil {
				return nil, err
			}
		}
		if !sameUUID(incident.AssigneeID, next) {
			changes = append(changes, changeEvent(incidentdomain.TimelineAssigneeChanged, actor, "assigneeId", incident.AssigneeID, next))
			incident.AssigneeID = next
		}
	}
	if len(changes) == 0 {
		return incident, nil
	}
	if err := s.repo.Update(ctx, incident, input.Version); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	for i := range changes {
		changes[i].IncidentID = incident.ID
		if err := s.repo.AppendTimeline(ctx, &changes[i]); err != nil {
			return nil, err
		}
		s.publish("timeline.appended", incident.ID, &changes[i])
	}
	s.publish("incident.updated", incident.ID, incident)
	return incident, nil
}

func (s *Service) Timeline(ctx context.Context, role authdomain.Role, id uuid.UUID) ([]*incidentdomain.TimelineEvent, error) {
	if _, err := s.Get(ctx, role, id); err != nil {
		return nil, err
	}
	return s.repo.Timeline(ctx, id)
}

func (s *Service) Comment(ctx context.Context, actor uuid.UUID, role authdomain.Role, id uuid.UUID, message string) (*incidentdomain.TimelineEvent, error) {
	if !role.AtLeast(authdomain.RoleResponder) {
		return nil, ErrForbidden
	}
	if _, err := s.Get(ctx, role, id); err != nil {
		return nil, err
	}
	message = strings.TrimSpace(message)
	if message == "" || len(message) > 5000 {
		return nil, fmt.Errorf("%w: comment must contain 1-5000 characters", ErrValidation)
	}
	event := &incidentdomain.TimelineEvent{IncidentID: id, Type: incidentdomain.TimelineCommented, ActorID: actor, Message: message}
	if err := s.repo.AppendTimeline(ctx, event); err != nil {
		return nil, err
	}
	s.publish("timeline.appended", id, event)
	return event, nil
}

func (s *Service) UploadAttachment(ctx context.Context, actor uuid.UUID, role authdomain.Role, incidentID uuid.UUID, stream io.Reader, size int64, filename, mimetype string) (*filedomain.File, error) {
	if !role.AtLeast(authdomain.RoleResponder) {
		return nil, ErrForbidden
	}
	if _, err := s.Get(ctx, role, incidentID); err != nil {
		return nil, err
	}
	if size <= 0 || size > s.maxFileSize {
		return nil, fmt.Errorf("%w: invalid attachment size", ErrValidation)
	}
	objectKey, err := s.storage.UploadFile(ctx, stream, size, filename, mimetype, actor)
	if err != nil {
		return nil, err
	}
	file := &filedomain.File{UserID: actor, OriginalName: filename, ObjectKey: objectKey, Size: size, Mimetype: mimetype, Bucket: s.bucket, Scope: "incident", IncidentID: &incidentID}
	if err := s.files.Create(ctx, file); err != nil {
		_ = s.storage.DeleteFile(ctx, objectKey)
		return nil, err
	}
	event := &incidentdomain.TimelineEvent{IncidentID: incidentID, Type: incidentdomain.TimelineAttachmentAdded, ActorID: actor, Message: filename, FileID: &file.ID}
	if err := s.repo.AppendTimeline(ctx, event); err != nil {
		return nil, err
	}
	s.publish("timeline.appended", incidentID, event)
	return file, nil
}

func (s *Service) Attachment(ctx context.Context, role authdomain.Role, incidentID, fileID uuid.UUID) (*filedomain.File, error) {
	if _, err := s.Get(ctx, role, incidentID); err != nil {
		return nil, err
	}
	file, err := s.files.GetIncidentAttachment(ctx, fileID, incidentID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrNotFound
	}
	return file, nil
}

func (s *Service) Events() (<-chan hub.Event, func()) { return s.hub.Subscribe() }

func (s *Service) publish(kind string, id uuid.UUID, data interface{}) {
	if s.hub != nil {
		s.hub.Publish(hub.Event{Type: kind, IncidentID: id.String(), Data: data})
	}
}

func (s *Service) validateAssignee(ctx context.Context, id uuid.UUID) error {
	if s.users == nil {
		return nil
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("%w: assignee not found", ErrValidation)
	}
	return nil
}

func severityFromHealth(status clusterdomain.HealthStatus) incidentdomain.Severity {
	switch status {
	case clusterdomain.StatusUnhealthy:
		return incidentdomain.SeverityCritical
	case clusterdomain.StatusDegraded:
		return incidentdomain.SeverityHigh
	case clusterdomain.StatusStarting:
		return incidentdomain.SeverityMedium
	default:
		return incidentdomain.SeverityLow
	}
}

func validTransition(from, to incidentdomain.Status) bool {
	rank := map[incidentdomain.Status]int{incidentdomain.StatusOpen: 0, incidentdomain.StatusInvestigating: 1, incidentdomain.StatusMitigated: 2, incidentdomain.StatusResolved: 3}
	return rank[to] > rank[from]
}

func changeEvent(kind incidentdomain.TimelineType, actor uuid.UUID, field string, from, to interface{}) incidentdomain.TimelineEvent {
	return incidentdomain.TimelineEvent{Type: kind, ActorID: actor, Changes: map[string]interface{}{field: map[string]interface{}{"from": from, "to": to}}}
}

func sameUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
