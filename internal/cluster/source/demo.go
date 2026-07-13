package source

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lab2/rest-api/internal/cluster/domain"
)

const (
	ScenarioLatency = "latency"
	ScenarioCrash   = "crash"
	ScenarioScale   = "scale"
	ScenarioRecover = "recover"
)

var ErrUnknownScenario = errors.New("unknown demo scenario")

type DemoSource struct {
	mu             sync.RWMutex
	snapshot       domain.Snapshot
	subscribers    map[chan domain.Event]struct{}
	readinessDelay time.Duration
	eventSequence  atomic.Uint64
}

func NewDemoSource(readinessDelay time.Duration) *DemoSource {
	if readinessDelay <= 0 {
		readinessDelay = 3800 * time.Millisecond
	}
	return &DemoSource{
		snapshot:       initialSnapshot(),
		subscribers:    make(map[chan domain.Event]struct{}),
		readinessDelay: readinessDelay,
	}
}

func (s *DemoSource) Snapshot(context.Context) (domain.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.snapshot), nil
}

func (s *DemoSource) Subscribe(ctx context.Context) (<-chan domain.Event, error) {
	updates := make(chan domain.Event, 8)
	s.mu.Lock()
	s.subscribers[updates] = struct{}{}
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		if _, ok := s.subscribers[updates]; ok {
			delete(s.subscribers, updates)
			close(updates)
		}
		s.mu.Unlock()
	}()

	return updates, nil
}

func (s *DemoSource) RunScenario(ctx context.Context, scenario string) (domain.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, err
	}

	s.mu.Lock()
	var event domain.Event
	switch scenario {
	case ScenarioLatency:
		event = s.applyLatency()
	case ScenarioCrash:
		event = s.applyCrash()
	case ScenarioScale:
		event = s.applyScale()
	case ScenarioRecover:
		event = s.applyRecovery()
	default:
		s.mu.Unlock()
		return domain.Snapshot{}, fmt.Errorf("%w: %s", ErrUnknownScenario, scenario)
	}
	s.prependEvent(event)
	snapshot := cloneSnapshot(s.snapshot)
	s.broadcastLocked(event)
	s.mu.Unlock()

	if scenario == ScenarioScale && event.Status == domain.StatusStarting {
		time.AfterFunc(s.readinessDelay, s.finishScale)
	}

	return snapshot, nil
}

func (s *DemoSource) applyLatency() domain.Event {
	for index := range s.snapshot.Services {
		service := &s.snapshot.Services[index]
		switch service.ID {
		case "redis":
			service.Status = domain.StatusDegraded
			service.Latency = 286
			service.ErrorRate = 2.84
			for instanceIndex := range service.Instances {
				service.Instances[instanceIndex].Status = domain.StatusDegraded
				service.Instances[instanceIndex].Latency = 286
			}
		case "api":
			service.Status = domain.StatusDegraded
			service.Latency = 148
			service.ErrorRate = 1.16
		}
	}
	return s.newEvent("redis", domain.StatusDegraded, "Latency threshold exceeded", "p95 reached 286 ms · Core API affected")
}

func (s *DemoSource) applyCrash() domain.Event {
	for index := range s.snapshot.Services {
		service := &s.snapshot.Services[index]
		if service.ID != "api" {
			continue
		}
		service.Status = domain.StatusDegraded
		service.Latency = 91
		service.ErrorRate = 4.72
		if len(service.Instances) > 0 {
			service.Instances[0].Status = domain.StatusUnhealthy
			service.Instances[0].Latency = 0
			service.Instances[0].Restarts++
		}
	}
	return s.newEvent("api", domain.StatusUnhealthy, "Pod stopped responding", "api-68f7d-k5jrm failed readiness probe")
}

func (s *DemoSource) applyScale() domain.Event {
	for index := range s.snapshot.Services {
		service := &s.snapshot.Services[index]
		if service.ID != "api" {
			continue
		}
		for _, instance := range service.Instances {
			if instance.ID == "api-68f7d-new01" {
				return s.newEvent("api", service.Status, "Scale already requested", "API replica is already present")
			}
		}
		service.Status = domain.StatusStarting
		service.Instances = append(service.Instances, domain.Instance{
			ID:        "api-68f7d-new01",
			Name:      "api-68f7d-new01",
			Status:    domain.StatusStarting,
			StartedAt: time.Now().UTC(),
		})
	}
	return s.newEvent("api", domain.StatusStarting, "Scaling deployment", "New API replica is waiting for readiness")
}

func (s *DemoSource) applyRecovery() domain.Event {
	previousEvents := append([]domain.Event(nil), s.snapshot.Events...)
	s.snapshot = initialSnapshot()
	s.snapshot.Events = previousEvents
	return s.newEvent("api", domain.StatusHealthy, "Cluster recovered", "All services returned to nominal state")
}

func (s *DemoSource) finishScale() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ready := false
	for index := range s.snapshot.Services {
		service := &s.snapshot.Services[index]
		if service.ID != "api" {
			continue
		}
		for instanceIndex := range service.Instances {
			instance := &service.Instances[instanceIndex]
			if instance.ID == "api-68f7d-new01" && instance.Status == domain.StatusStarting {
				instance.Status = domain.StatusHealthy
				instance.Latency = 29
				ready = true
			}
		}
		if ready {
			service.Status = domain.StatusHealthy
			service.Latency = 27
			for _, instance := range service.Instances {
				if instance.Status == domain.StatusUnhealthy || instance.Status == domain.StatusDegraded {
					service.Status = domain.StatusDegraded
					break
				}
			}
		}
	}
	if !ready {
		return
	}

	event := s.newEvent("api", domain.StatusHealthy, "Replica is ready", "api-68f7d-new01 joined the service mesh")
	s.prependEvent(event)
	s.broadcastLocked(event)
}

func (s *DemoSource) newEvent(serviceID string, status domain.HealthStatus, title, detail string) domain.Event {
	return domain.Event{
		ID:        fmt.Sprintf("demo-%d-%d", time.Now().UnixMilli(), s.eventSequence.Add(1)),
		ServiceID: serviceID,
		Status:    status,
		Title:     title,
		Detail:    detail,
		Timestamp: time.Now().UTC(),
	}
}

func (s *DemoSource) prependEvent(event domain.Event) {
	s.snapshot.GeneratedAt = time.Now().UTC()
	s.snapshot.Events = append([]domain.Event{event}, s.snapshot.Events...)
	if len(s.snapshot.Events) > 12 {
		s.snapshot.Events = s.snapshot.Events[:12]
	}
}

func (s *DemoSource) broadcastLocked(event domain.Event) {
	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func initialSnapshot() domain.Snapshot {
	startedAt := time.Now().UTC().Add(-18 * 24 * time.Hour)
	instance := func(name string, latency int) domain.Instance {
		return domain.Instance{ID: name, Name: name, Status: domain.StatusHealthy, Latency: latency, StartedAt: startedAt}
	}

	return domain.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Services: []domain.Service{
			{ID: "gateway", Name: "Edge Gateway", Description: "Public ingress and request routing", Kind: "edge", Status: domain.StatusHealthy, Position: [3]float64{-2.45, 0.65, 1.05}, Latency: 18, Uptime: 99.99, RequestsPerMinute: 1842, ErrorRate: 0.03, Version: "nginx/1.27", Instances: []domain.Instance{instance("gateway-7db9f-2rk8m", 17), instance("gateway-7db9f-x9tq4", 19)}, Dependencies: []string{"api"}},
			{ID: "api", Name: "Core API", Description: "Authentication, incidents and cluster topology", Kind: "compute", Status: domain.StatusHealthy, Position: [3]float64{-0.85, 0.55, 0.2}, Latency: 32, Uptime: 99.97, RequestsPerMinute: 1396, ErrorRate: 0.08, Version: "neuro-api/0.1.0", Instances: []domain.Instance{instance("api-68f7d-k5jrm", 30), instance("api-68f7d-p82nc", 34)}, Dependencies: []string{"mongodb", "redis", "rabbitmq", "minio"}},
			{ID: "mongodb", Name: "MongoDB", Description: "Persistent cluster state and incident history", Kind: "database", Status: domain.StatusHealthy, Position: [3]float64{1.35, 1.35, -0.85}, Latency: 24, Uptime: 99.98, RequestsPerMinute: 884, ErrorRate: 0.01, Version: "mongo/7.0", Instances: []domain.Instance{instance("mongodb-0", 24)}, Dependencies: []string{}},
			{ID: "redis", Name: "Redis", Description: "Snapshots, cache and distributed locks", Kind: "cache", Status: domain.StatusHealthy, Position: [3]float64{1.65, 0.05, 0.65}, Latency: 7, Uptime: 99.99, RequestsPerMinute: 2234, ErrorRate: 0, Version: "redis/7.2", Instances: []domain.Instance{instance("redis-6dc8b-m2v7s", 7)}, Dependencies: []string{}},
			{ID: "rabbitmq", Name: "RabbitMQ", Description: "Cluster events and notification delivery", Kind: "queue", Status: domain.StatusHealthy, Position: [3]float64{-0.35, -1.35, -0.9}, Latency: 12, Uptime: 99.96, RequestsPerMinute: 428, ErrorRate: 0.02, Version: "rabbitmq/3.12", Instances: []domain.Instance{instance("rabbitmq-0", 12)}, Dependencies: []string{}},
			{ID: "minio", Name: "MinIO", Description: "Logs, screenshots and incident attachments", Kind: "storage", Status: domain.StatusHealthy, Position: [3]float64{1.55, -1.25, 0.9}, Latency: 41, Uptime: 99.95, RequestsPerMinute: 96, ErrorRate: 0.04, Version: "minio/2025.01", Instances: []domain.Instance{instance("minio-0", 41)}, Dependencies: []string{}},
		},
		Events: []domain.Event{
			{ID: "boot-1", ServiceID: "api", Status: domain.StatusHealthy, Title: "Topology synchronized", Detail: "6 services · 8 instances discovered", Timestamp: time.Now().UTC().Add(-18 * time.Second)},
			{ID: "boot-2", ServiceID: "gateway", Status: domain.StatusHealthy, Title: "Health probes stable", Detail: "All readiness checks are passing", Timestamp: time.Now().UTC().Add(-52 * time.Second)},
		},
	}
}

func cloneSnapshot(snapshot domain.Snapshot) domain.Snapshot {
	clone := snapshot
	clone.Services = make([]domain.Service, len(snapshot.Services))
	for index, service := range snapshot.Services {
		clone.Services[index] = service
		clone.Services[index].Instances = append([]domain.Instance(nil), service.Instances...)
		clone.Services[index].Dependencies = append([]string(nil), service.Dependencies...)
	}
	clone.Events = append([]domain.Event(nil), snapshot.Events...)
	return clone
}
