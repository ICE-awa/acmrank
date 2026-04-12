package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrIntegrationAlertNotFound = errors.New("integration alert not found")

type integrationAlertRepositoryDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type RecordIntegrationAlertParams struct {
	Integration model.IntegrationName
	Summary     string
	Detail      string
	TriggeredAt time.Time
}

type IntegrationAlertRepository struct {
	db integrationAlertRepositoryDB
}

func NewIntegrationAlertRepository(db integrationAlertRepositoryDB) *IntegrationAlertRepository {
	return &IntegrationAlertRepository{db: db}
}

func (r *IntegrationAlertRepository) RecordFailure(
	ctx context.Context,
	params RecordIntegrationAlertParams,
) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO integration_alerts (
  integration, status, summary, detail, first_triggered_at, last_triggered_at, resolved_at, failure_count
)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $5, NULL, 1)
ON CONFLICT (integration) DO UPDATE
SET status = EXCLUDED.status,
    summary = EXCLUDED.summary,
    detail = EXCLUDED.detail,
    last_triggered_at = EXCLUDED.last_triggered_at,
    resolved_at = NULL,
    failure_count = integration_alerts.failure_count + 1,
    updated_at = NOW()`,
		params.Integration,
		model.IntegrationAlertStatusActive,
		params.Summary,
		params.Detail,
		params.TriggeredAt.UTC(),
	)
	return err
}

func (r *IntegrationAlertRepository) Resolve(
	ctx context.Context,
	integration model.IntegrationName,
	resolvedAt time.Time,
) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE integration_alerts
SET status = $2,
    resolved_at = $3,
    updated_at = $3
WHERE integration = $1`,
		integration,
		model.IntegrationAlertStatusResolved,
		resolvedAt.UTC(),
	)
	return err
}

func (r *IntegrationAlertRepository) GetByIntegration(
	ctx context.Context,
	integration model.IntegrationName,
) (model.IntegrationAlert, error) {
	var item model.IntegrationAlert
	var resolvedAt *time.Time
	err := r.db.QueryRow(
		ctx,
		`SELECT id, integration, status, summary, COALESCE(detail, ''), first_triggered_at, last_triggered_at,
       resolved_at, failure_count, created_at, updated_at
FROM integration_alerts
WHERE integration = $1`,
		integration,
	).Scan(
		&item.ID,
		&item.Integration,
		&item.Status,
		&item.Summary,
		&item.Detail,
		&item.FirstTriggeredAt,
		&item.LastTriggeredAt,
		&resolvedAt,
		&item.FailureCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.IntegrationAlert{}, ErrIntegrationAlertNotFound
		}

		return model.IntegrationAlert{}, err
	}

	item.ResolvedAt = resolvedAt
	return item, nil
}
