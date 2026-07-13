package domain

import (
	"time"

	"github.com/google/uuid"
	clusterdomain "github.com/lab2/rest-api/internal/cluster/domain"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

func (s Severity) Valid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusOpen          Status = "open"
	StatusInvestigating Status = "investigating"
	StatusMitigated     Status = "mitigated"
	StatusResolved      Status = "resolved"
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusInvestigating, StatusMitigated, StatusResolved:
		return true
	default:
		return false
	}
}

type Incident struct {
	ID          uuid.UUID            `bson:"_id" json:"id"`
	Title       string               `bson:"title" json:"title"`
	Description string               `bson:"description" json:"description"`
	Service     string               `bson:"service" json:"service"`
	Severity    Severity             `bson:"severity" json:"severity"`
	Status      Status               `bson:"status" json:"status"`
	AssigneeID  *uuid.UUID           `bson:"assignee_id,omitempty" json:"assigneeId,omitempty"`
	CreatorID   uuid.UUID            `bson:"creator_id" json:"creatorId"`
	SourceEvent *clusterdomain.Event `bson:"source_event,omitempty" json:"sourceEvent,omitempty"`
	Version     int64                `bson:"version" json:"version"`
	CreatedAt   time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updated_at" json:"updatedAt"`
}

type TimelineType string

const (
	TimelineCreated         TimelineType = "created"
	TimelineCommented       TimelineType = "commented"
	TimelineStatusChanged   TimelineType = "status.changed"
	TimelineSeverityChanged TimelineType = "severity.changed"
	TimelineAssigneeChanged TimelineType = "assignee.changed"
	TimelineIncidentEdited  TimelineType = "incident.edited"
	TimelineAttachmentAdded TimelineType = "attachment.added"
)

type TimelineEvent struct {
	ID         uuid.UUID              `bson:"_id" json:"id"`
	IncidentID uuid.UUID              `bson:"incident_id" json:"incidentId"`
	Type       TimelineType           `bson:"type" json:"type"`
	ActorID    uuid.UUID              `bson:"actor_id" json:"actorId"`
	Message    string                 `bson:"message,omitempty" json:"message,omitempty"`
	Changes    map[string]interface{} `bson:"changes,omitempty" json:"changes,omitempty"`
	FileID     *uuid.UUID             `bson:"file_id,omitempty" json:"fileId,omitempty"`
	CreatedAt  time.Time              `bson:"created_at" json:"createdAt"`
}

type Filter struct {
	Status     Status
	Severity   Severity
	Service    string
	AssigneeID *uuid.UUID
	Page       int64
	Limit      int64
}

type Page struct {
	Data       []*Incident `json:"data"`
	Page       int64       `json:"page"`
	Limit      int64       `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int64       `json:"totalPages"`
}
