package source

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lab2/rest-api/internal/cluster/domain"
)

type ClusterSource interface {
	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Subscribe(ctx context.Context) (<-chan domain.Event, error)
}

type ScenarioRunner interface {
	RunScenario(ctx context.Context, scenario string) (domain.Snapshot, error)
}

var ErrUnknownSource = errors.New("unknown cluster source")

// ErrUnknownClusterSource is kept as a descriptive alias for callers that
// expose source selection errors directly.
var ErrUnknownClusterSource = ErrUnknownSource

// NewFactory creates the configured cluster source. Kubernetes support is
// intentionally left for a later iteration; demo is the only source enabled
// in this release.
func NewFactory(name string, readinessDelay time.Duration) (ClusterSource, ScenarioRunner, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "demo":
		demo := NewDemoSource(readinessDelay)
		return demo, demo, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnknownSource, name)
	}
}

// New is a short compatibility alias for NewFactory.
func New(name string, readinessDelay time.Duration) (ClusterSource, ScenarioRunner, error) {
	return NewFactory(name, readinessDelay)
}

// NewClusterSource is an explicit alias for NewFactory.
func NewClusterSource(name string, readinessDelay time.Duration) (ClusterSource, ScenarioRunner, error) {
	return NewFactory(name, readinessDelay)
}
