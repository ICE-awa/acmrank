package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrSyncJobNotFound  = errors.New("sync job not found")
	ErrNoPendingSyncJob = errors.New("no pending sync job")
)

type syncJobRepositoryDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type syncJobTx interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type pgxSyncJobTx struct {
	tx pgx.Tx
}

func (t pgxSyncJobTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t pgxSyncJobTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxSyncJobTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type EnqueueSyncJobParams struct {
	SiteUserID        int64
	PlatformAccountID int64
	Platform          string
	JobType           model.SyncJobType
	ScheduledAt       time.Time
}

type SyncJobRepository struct {
	db      syncJobRepositoryDB
	beginTx func(context.Context) (syncJobTx, error)
}

func NewSyncJobRepository(db syncJobRepositoryDB) *SyncJobRepository {
	return &SyncJobRepository{
		db: db,
		beginTx: func(ctx context.Context) (syncJobTx, error) {
			tx, err := db.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return nil, err
			}

			return pgxSyncJobTx{tx: tx}, nil
		},
	}
}

func (r *SyncJobRepository) Enqueue(
	ctx context.Context,
	params EnqueueSyncJobParams,
) (model.SyncJob, error) {
	activeJob, err := scanSyncJob(
		r.db.QueryRow(
			ctx,
			`SELECT id, site_user_id, platform_account_id, COALESCE(platform, ''), job_type, status,
       scheduled_at, started_at, finished_at, attempt_count, COALESCE(error_message, ''),
       created_at, updated_at
FROM sync_jobs
WHERE platform_account_id = $1
  AND job_type = $2
  AND status IN ('queued', 'running')
ORDER BY created_at DESC, id DESC
LIMIT 1`,
			params.PlatformAccountID,
			params.JobType,
		),
	)
	if err == nil {
		return activeJob, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.SyncJob{}, err
	}

	job, err := scanSyncJob(
		r.db.QueryRow(
			ctx,
			`INSERT INTO sync_jobs (
  site_user_id, platform_account_id, platform, job_type, status, scheduled_at
)
VALUES ($1, $2, NULLIF($3, ''), $4, 'queued', $5)
RETURNING id, site_user_id, platform_account_id, COALESCE(platform, ''), job_type, status,
          scheduled_at, started_at, finished_at, attempt_count, COALESCE(error_message, ''),
          created_at, updated_at`,
			params.SiteUserID,
			params.PlatformAccountID,
			params.Platform,
			params.JobType,
			params.ScheduledAt.UTC(),
		),
	)
	if err != nil {
		return model.SyncJob{}, err
	}

	return job, nil
}

func (r *SyncJobRepository) ClaimNextQueuedJob(
	ctx context.Context,
	jobType model.SyncJobType,
	startedAt time.Time,
) (model.SyncJob, error) {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return model.SyncJob{}, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	job, err := scanSyncJob(
		tx.QueryRow(
			ctx,
			`WITH next_job AS (
  SELECT id
  FROM sync_jobs
  WHERE job_type = $1
    AND status = 'queued'
    AND scheduled_at <= $2
  ORDER BY scheduled_at ASC, id ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE sync_jobs AS jobs
SET status = 'running',
    started_at = $2,
    finished_at = NULL,
    error_message = NULL,
    attempt_count = jobs.attempt_count + 1,
    updated_at = $2
FROM next_job
WHERE jobs.id = next_job.id
RETURNING jobs.id, jobs.site_user_id, jobs.platform_account_id, COALESCE(jobs.platform, ''),
          jobs.job_type, jobs.status, jobs.scheduled_at, jobs.started_at, jobs.finished_at,
          jobs.attempt_count, COALESCE(jobs.error_message, ''), jobs.created_at, jobs.updated_at`,
			jobType,
			startedAt.UTC(),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.SyncJob{}, ErrNoPendingSyncJob
		}

		return model.SyncJob{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.SyncJob{}, err
	}

	committed = true
	return job, nil
}

func (r *SyncJobRepository) MarkSucceeded(
	ctx context.Context,
	jobID int64,
	finishedAt time.Time,
) error {
	return r.updateJobCompletion(
		ctx,
		jobID,
		model.SyncJobStatusSucceeded,
		"",
		finishedAt,
	)
}

func (r *SyncJobRepository) MarkFailed(
	ctx context.Context,
	jobID int64,
	errorMessage string,
	finishedAt time.Time,
) error {
	return r.updateJobCompletion(
		ctx,
		jobID,
		model.SyncJobStatusFailed,
		errorMessage,
		finishedAt,
	)
}

func (r *SyncJobRepository) updateJobCompletion(
	ctx context.Context,
	jobID int64,
	status model.SyncJobStatus,
	errorMessage string,
	finishedAt time.Time,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`UPDATE sync_jobs
SET status = $2,
    finished_at = $3,
    error_message = NULLIF($4, ''),
    updated_at = $3
WHERE id = $1`,
		jobID,
		status,
		finishedAt.UTC(),
		errorMessage,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrSyncJobNotFound
	}

	return nil
}

type syncJobScanner interface {
	Scan(dest ...any) error
}

func scanSyncJob(scanner syncJobScanner) (model.SyncJob, error) {
	var job model.SyncJob
	var siteUserID *int64
	var platformAccountID *int64
	var startedAt *time.Time
	var finishedAt *time.Time

	err := scanner.Scan(
		&job.ID,
		&siteUserID,
		&platformAccountID,
		&job.Platform,
		&job.JobType,
		&job.Status,
		&job.ScheduledAt,
		&startedAt,
		&finishedAt,
		&job.AttemptCount,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return model.SyncJob{}, err
	}

	job.SiteUserID = siteUserID
	job.PlatformAccountID = platformAccountID
	job.StartedAt = startedAt
	job.FinishedAt = finishedAt
	return job, nil
}
