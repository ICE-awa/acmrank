package service

import (
	"context"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type stubHealthReader struct {
	snapshot model.HealthSnapshot
	err      error
}

func (s stubHealthReader) Snapshot(context.Context) (model.HealthSnapshot, error) {
	return s.snapshot, s.err
}

func TestHealthServiceReturnsOKWhenDependenciesAreReachable(t *testing.T) {
	svc := NewHealthService(stubHealthReader{
		snapshot: model.HealthSnapshot{
			Service:   "api",
			Version:   "dev",
			StartedAt: time.Now().UTC(),
			CheckedAt: time.Now().UTC(),
			Dependencies: []model.DependencyHealth{
				{Name: "postgres", Configured: true, Reachable: true},
				{Name: "redis", Configured: true, Reachable: true},
				{Name: "nats", Configured: true, Reachable: true},
			},
		},
	})

	report, err := svc.GetReport(context.Background())
	if err != nil {
		t.Fatalf("GetReport() error = %v", err)
	}

	if report.Status != model.HealthStatusOK {
		t.Fatalf("GetReport() status = %q, want %q", report.Status, model.HealthStatusOK)
	}
}

func TestHealthServiceReturnsDegradedWhenDependencyFails(t *testing.T) {
	svc := NewHealthService(stubHealthReader{
		snapshot: model.HealthSnapshot{
			Service:   "scheduler",
			Version:   "dev",
			StartedAt: time.Now().UTC(),
			CheckedAt: time.Now().UTC(),
			Dependencies: []model.DependencyHealth{
				{Name: "postgres", Configured: true, Reachable: true},
				{Name: "redis", Configured: true, Reachable: false},
			},
		},
	})

	report, err := svc.GetReport(context.Background())
	if err != nil {
		t.Fatalf("GetReport() error = %v", err)
	}

	if report.Status != model.HealthStatusDegraded {
		t.Fatalf("GetReport() status = %q, want %q", report.Status, model.HealthStatusDegraded)
	}
}
