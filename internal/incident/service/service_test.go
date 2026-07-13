package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authdomain "github.com/lab2/rest-api/internal/auth/domain"
	clusterdomain "github.com/lab2/rest-api/internal/cluster/domain"
	incidentdomain "github.com/lab2/rest-api/internal/incident/domain"
	"github.com/lab2/rest-api/internal/incident/hub"
	incidentrepository "github.com/lab2/rest-api/internal/incident/repository"
)

type memoryRepository struct {
	incident *incidentdomain.Incident
	events   []*incidentdomain.TimelineEvent
}

func (m *memoryRepository) Create(_ context.Context, incident *incidentdomain.Incident, event *incidentdomain.TimelineEvent) error {
	incident.ID = uuid.New()
	incident.Version = 1
	incident.CreatedAt = time.Now()
	incident.UpdatedAt = incident.CreatedAt
	event.IncidentID = incident.ID
	event.ID = uuid.New()
	m.incident = incident
	m.events = append(m.events, event)
	return nil
}
func (m *memoryRepository) Get(_ context.Context, id uuid.UUID) (*incidentdomain.Incident, error) {
	if m.incident == nil || m.incident.ID != id {
		return nil, nil
	}
	copy := *m.incident
	return &copy, nil
}
func (m *memoryRepository) List(context.Context, incidentdomain.Filter) (*incidentdomain.Page, error) {
	return &incidentdomain.Page{Data: []*incidentdomain.Incident{m.incident}}, nil
}
func (m *memoryRepository) Update(_ context.Context, incident *incidentdomain.Incident, expected int64) error {
	if m.incident.Version != expected {
		return incidentrepository.ErrVersionConflict
	}
	*m.incident = *incident
	m.incident.Version = expected + 1
	return nil
}
func (m *memoryRepository) AppendTimeline(_ context.Context, event *incidentdomain.TimelineEvent) error {
	event.ID = uuid.New()
	m.events = append(m.events, event)
	return nil
}
func (m *memoryRepository) Timeline(context.Context, uuid.UUID) ([]*incidentdomain.TimelineEvent, error) {
	return m.events, nil
}

type memoryCluster struct{}

func (memoryCluster) Snapshot(context.Context) (clusterdomain.Snapshot, error) {
	return clusterdomain.Snapshot{}, nil
}
func (memoryCluster) Subscribe(context.Context) (<-chan clusterdomain.Event, error) {
	return make(chan clusterdomain.Event), nil
}

func TestIncidentRBACAndOptimisticLifecycle(t *testing.T) {
	repo := &memoryRepository{}
	svc := New(repo, memoryCluster{}, hub.New(), nil, nil, "", 1024)
	ctx := context.Background()
	viewer := uuid.New()
	responder := uuid.New()
	manager := uuid.New()
	if _, err := svc.Create(ctx, viewer, authdomain.RoleViewer, CreateInput{Title: "Nope", Service: "api"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer create error = %v", err)
	}
	incident, err := svc.Create(ctx, responder, authdomain.RoleResponder, CreateInput{Title: "API outage", Service: "api"})
	if err != nil {
		t.Fatal(err)
	}
	staleVersion := incident.Version
	resolved := incidentdomain.StatusResolved
	if _, err := svc.Patch(ctx, responder, authdomain.RoleResponder, incident.ID, PatchInput{Version: incident.Version, Status: &resolved}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("responder resolve error = %v", err)
	}
	investigating := incidentdomain.StatusInvestigating
	if _, err := svc.Patch(ctx, manager, authdomain.RoleIncidentManager, incident.ID, PatchInput{Version: staleVersion, Status: &investigating}); err != nil {
		t.Fatal(err)
	}
	mitigated := incidentdomain.StatusMitigated
	if _, err := svc.Patch(ctx, manager, authdomain.RoleIncidentManager, incident.ID, PatchInput{Version: staleVersion, Status: &mitigated}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale version error = %v", err)
	}
}

var _ incidentrepository.Repository = (*memoryRepository)(nil)
