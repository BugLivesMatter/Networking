package source

import (
	"context"
	"testing"
	"time"

	"github.com/lab2/rest-api/internal/cluster/domain"
)

func TestDemoSourceInitialSnapshot(t *testing.T) {
	source := NewDemoSource(time.Second)

	snapshot, err := source.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(snapshot.Services) != 6 {
		t.Fatalf("services = %d, want 6", len(snapshot.Services))
	}

	instances := 0
	for _, service := range snapshot.Services {
		instances += len(service.Instances)
		if service.Status != domain.StatusHealthy {
			t.Errorf("service %q status = %q, want healthy", service.ID, service.Status)
		}
	}
	if instances != 8 {
		t.Errorf("instances = %d, want 8", instances)
	}
}

func TestDemoSourceLatencyDegradesDependencyAndAPI(t *testing.T) {
	source := NewDemoSource(time.Second)

	snapshot, err := source.RunScenario(context.Background(), ScenarioLatency)
	if err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}

	redis := serviceByID(t, snapshot, "redis")
	api := serviceByID(t, snapshot, "api")
	if redis.Status != domain.StatusDegraded || redis.Latency != 286 {
		t.Errorf("redis = (%q, %dms), want (degraded, 286ms)", redis.Status, redis.Latency)
	}
	if api.Status != domain.StatusDegraded {
		t.Errorf("api status = %q, want degraded", api.Status)
	}
	if snapshot.Events[0].ServiceID != "redis" {
		t.Errorf("latest event service = %q, want redis", snapshot.Events[0].ServiceID)
	}
}

func TestDemoSourceScalePublishesStartingAndReady(t *testing.T) {
	source := NewDemoSource(10 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := source.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if _, err := source.RunScenario(ctx, ScenarioScale); err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}

	assertEventStatus(t, events, domain.StatusStarting)
	assertEventStatus(t, events, domain.StatusHealthy)

	snapshot, err := source.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	api := serviceByID(t, snapshot, "api")
	if api.Status != domain.StatusHealthy {
		t.Errorf("api status = %q, want healthy", api.Status)
	}
	if len(api.Instances) != 3 {
		t.Errorf("api instances = %d, want 3", len(api.Instances))
	}
}

func TestDemoSourceRejectsUnknownScenario(t *testing.T) {
	source := NewDemoSource(time.Second)
	if _, err := source.RunScenario(context.Background(), "explode"); err == nil {
		t.Fatal("RunScenario() error = nil, want error")
	}
}

func serviceByID(t *testing.T, snapshot domain.Snapshot, id string) domain.Service {
	t.Helper()
	for _, service := range snapshot.Services {
		if service.ID == id {
			return service
		}
	}
	t.Fatalf("service %q not found", id)
	return domain.Service{}
}

func assertEventStatus(t *testing.T, events <-chan domain.Event, status domain.HealthStatus) {
	t.Helper()
	select {
	case event := <-events:
		if event.Status != status {
			t.Errorf("event status = %q, want %q", event.Status, status)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %q event", status)
	}
}
