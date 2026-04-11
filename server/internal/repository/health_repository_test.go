package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/model"
)

type stubDependencySource struct {
	startedAt time.Time
	statuses  [][]model.DependencyHealth
	calls     int
}

func (s *stubDependencySource) StartedAt() time.Time {
	return s.startedAt
}

func (s *stubDependencySource) Statuses(context.Context) []model.DependencyHealth {
	current := s.statuses[s.calls]
	s.calls++

	copied := make([]model.DependencyHealth, len(current))
	copy(copied, current)
	return copied
}

func TestHealthRepositorySnapshotsCurrentDependencyState(t *testing.T) {
	t.Parallel()

	source := &stubDependencySource{
		startedAt: time.Unix(100, 0).UTC(),
		statuses: [][]model.DependencyHealth{
			{{Name: "postgres", Configured: true, Reachable: true, Message: "connected"}},
			{{Name: "postgres", Configured: true, Reachable: false, Message: "ping failed"}},
		},
	}

	repository := NewHealthRepository(appmeta.ServiceAPI, "dev", source)

	first, err := repository.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() first error = %v", err)
	}

	second, err := repository.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() second error = %v", err)
	}

	if !first.Dependencies[0].Reachable {
		t.Fatal("Snapshot() first dependencies should use the current healthy status")
	}

	if second.Dependencies[0].Reachable {
		t.Fatal("Snapshot() second dependencies reused stale health state")
	}
}
