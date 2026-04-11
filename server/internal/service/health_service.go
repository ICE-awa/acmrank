package service

import (
	"context"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type HealthReader interface {
	Snapshot(context.Context) (model.HealthSnapshot, error)
}

type HealthService struct {
	reader HealthReader
}

func NewHealthService(reader HealthReader) *HealthService {
	return &HealthService{reader: reader}
}

func (s *HealthService) GetReport(ctx context.Context) (model.HealthReport, error) {
	snapshot, err := s.reader.Snapshot(ctx)
	if err != nil {
		return model.HealthReport{}, err
	}

	report := model.HealthReport{
		Service:      snapshot.Service,
		Version:      snapshot.Version,
		Status:       evaluateHealthStatus(snapshot.Dependencies),
		StartedAt:    snapshot.StartedAt,
		CheckedAt:    snapshot.CheckedAt,
		Dependencies: snapshot.Dependencies,
	}

	return report, nil
}

func evaluateHealthStatus(dependencies []model.DependencyHealth) model.HealthStatus {
	for _, dependency := range dependencies {
		if !dependency.Configured || !dependency.Reachable {
			return model.HealthStatusDegraded
		}
	}

	return model.HealthStatusOK
}
