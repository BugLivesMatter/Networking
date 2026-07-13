package domain

import "time"

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusStarting  HealthStatus = "starting"
	StatusUnknown   HealthStatus = "unknown"
)

type Instance struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Status    HealthStatus `json:"status"`
	Latency   int          `json:"latency"`
	Restarts  int          `json:"restarts"`
	StartedAt time.Time    `json:"startedAt"`
}

type Service struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Kind              string       `json:"kind"`
	Status            HealthStatus `json:"status"`
	Position          [3]float64   `json:"position"`
	Latency           int          `json:"latency"`
	Uptime            float64      `json:"uptime"`
	RequestsPerMinute int          `json:"requestsPerMinute"`
	ErrorRate         float64      `json:"errorRate"`
	Version           string       `json:"version"`
	Instances         []Instance   `json:"instances"`
	Dependencies      []string     `json:"dependencies"`
}

type Event struct {
	ID        string       `json:"id"`
	ServiceID string       `json:"serviceId"`
	Status    HealthStatus `json:"status"`
	Title     string       `json:"title"`
	Detail    string       `json:"detail"`
	Timestamp time.Time    `json:"timestamp"`
}

type Snapshot struct {
	Services    []Service `json:"services"`
	Events      []Event   `json:"events"`
	GeneratedAt time.Time `json:"generatedAt"`
}
