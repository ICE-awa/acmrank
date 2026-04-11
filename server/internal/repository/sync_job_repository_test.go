package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubSyncJobDB struct {
	queryRowFn func(context.Context, string, ...any) pgx.Row
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *stubSyncJobDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, query, args...)
}

func (s *stubSyncJobDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if s.execFn != nil {
		return s.execFn(ctx, query, args...)
	}

	return pgconn.NewCommandTag("UPDATE 0"), nil
}

func (s *stubSyncJobDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("unexpected BeginTx call")
}

type stubSyncJobTx struct {
	queryRowFn func(context.Context, string, ...any) pgx.Row
	commitFn   func(context.Context) error
	rollbackFn func(context.Context) error
}

func (s stubSyncJobTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, query, args...)
}

func (s stubSyncJobTx) Commit(ctx context.Context) error {
	if s.commitFn != nil {
		return s.commitFn(ctx)
	}

	return nil
}

func (s stubSyncJobTx) Rollback(ctx context.Context) error {
	if s.rollbackFn != nil {
		return s.rollbackFn(ctx)
	}

	return nil
}

func TestSyncJobRepositoryEnqueueReturnsExistingActiveJob(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_700_000, 0).UTC()
	accountID := int64(8)
	siteUserID := int64(7)
	repository := NewSyncJobRepository(&stubSyncJobDB{
		queryRowFn: func(_ context.Context, query string, args ...any) pgx.Row {
			if !strings.Contains(query, "status IN ('queued', 'running')") {
				t.Fatalf("unexpected query: %q", query)
			}

			return stubRow{
				scanFn: func(dest ...any) error {
					assignSyncJobRow(dest, model.SyncJob{
						ID:                11,
						SiteUserID:        &siteUserID,
						PlatformAccountID: &accountID,
						Platform:          "codeforces",
						JobType:           model.SyncJobTypeCodeforces,
						Status:            model.SyncJobStatusQueued,
						ScheduledAt:       now,
						CreatedAt:         now,
						UpdatedAt:         now,
					})
					return nil
				},
			}
		},
	})

	job, err := repository.Enqueue(context.Background(), EnqueueSyncJobParams{
		SiteUserID:        7,
		PlatformAccountID: 8,
		Platform:          "codeforces",
		JobType:           model.SyncJobTypeCodeforces,
		ScheduledAt:       now,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if job.ID != 11 || job.Status != model.SyncJobStatusQueued {
		t.Fatalf("Enqueue() job = %+v", job)
	}
}

func TestSyncJobRepositoryClaimNextQueuedJobMapsEmptyQueue(t *testing.T) {
	t.Parallel()

	repository := NewSyncJobRepository(&stubSyncJobDB{
		queryRowFn: func(context.Context, string, ...any) pgx.Row { return stubRow{} },
	})
	repository.beginTx = func(context.Context) (syncJobTx, error) {
		return stubSyncJobTx{
			queryRowFn: func(context.Context, string, ...any) pgx.Row {
				return stubRow{
					scanFn: func(dest ...any) error { return pgx.ErrNoRows },
				}
			},
		}, nil
	}

	_, err := repository.ClaimNextQueuedJob(context.Background(), model.SyncJobTypeCodeforces, time.Now())
	if !errors.Is(err, ErrNoPendingSyncJob) {
		t.Fatalf("ClaimNextQueuedJob() error = %v, want %v", err, ErrNoPendingSyncJob)
	}
}

func assignSyncJobRow(dest []any, job model.SyncJob) {
	*(dest[0].(*int64)) = job.ID
	*(dest[1].(**int64)) = job.SiteUserID
	*(dest[2].(**int64)) = job.PlatformAccountID
	*(dest[3].(*string)) = job.Platform
	*(dest[4].(*model.SyncJobType)) = job.JobType
	*(dest[5].(*model.SyncJobStatus)) = job.Status
	*(dest[6].(*time.Time)) = job.ScheduledAt
	*(dest[7].(**time.Time)) = job.StartedAt
	*(dest[8].(**time.Time)) = job.FinishedAt
	*(dest[9].(*int)) = job.AttemptCount
	*(dest[10].(*string)) = job.ErrorMessage
	*(dest[11].(*time.Time)) = job.CreatedAt
	*(dest[12].(*time.Time)) = job.UpdatedAt
}
