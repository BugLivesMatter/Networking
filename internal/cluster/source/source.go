package source

import (
	"context"

	"github.com/lab2/rest-api/internal/cluster/domain"
)

type ClusterSource interface {
	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Subscribe(ctx context.Context) (<-chan domain.Event, error)
}

type ScenarioRunner interface {
	RunScenario(ctx context.Context, scenario string) (domain.Snapshot, error)
}
