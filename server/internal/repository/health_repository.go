package repository

import (
	"context"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/model"
)

type HealthDependencySource interface {
	StartedAt() time.Time
	Statuses(context.Context) []model.DependencyHealth
}

type HealthRepository struct {
	service appmeta.ServiceName
	version string
	source  HealthDependencySource
}

func NewHealthRepository(
	service appmeta.ServiceName,
	version string,
	source HealthDependencySource,
) *HealthRepository {
	return &HealthRepository{
		service: service,
		version: version,
		source:  source,
	}
}

func (r *HealthRepository) Snapshot(ctx context.Context) (model.HealthSnapshot, error) {
	return model.HealthSnapshot{
		Service:      string(r.service),
		Version:      r.version,
		StartedAt:    r.source.StartedAt(),
		CheckedAt:    time.Now().UTC(),
		Dependencies: r.source.Statuses(ctx),
	}, nil
}
