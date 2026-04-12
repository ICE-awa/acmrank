package service

import (
	"context"
	"testing"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type stubPlatformSyncAccountStore struct {
	getByIDFn func(context.Context, int64) (model.PlatformAccount, error)
}

func (s stubPlatformSyncAccountStore) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	return s.getByIDFn(ctx, accountID)
}

type stubPlatformSyncEnqueuer struct {
	enqueueSyncFn func(context.Context, int64, int64) (model.SyncJob, error)
}

func (s stubPlatformSyncEnqueuer) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	return s.enqueueSyncFn(ctx, siteUserID, accountID)
}

func TestPlatformSyncServiceDispatchesToLuogu(t *testing.T) {
	t.Parallel()

	service := NewPlatformSyncService(
		stubPlatformSyncAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformLuogu,
				}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
				t.Fatal("atcoder enqueuer should not be called")
				return model.SyncJob{}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
				t.Fatal("codeforces enqueuer should not be called")
				return model.SyncJob{}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(_ context.Context, siteUserID int64, accountID int64) (model.SyncJob, error) {
				if siteUserID != 7 || accountID != 8 {
					t.Fatalf("EnqueueSync() siteUserID=%d accountID=%d", siteUserID, accountID)
				}

				return model.SyncJob{ID: 1, JobType: model.SyncJobTypeLuogu}, nil
			},
		},
	)

	job, err := service.EnqueueSync(context.Background(), 7, 8)
	if err != nil {
		t.Fatalf("EnqueueSync() error = %v", err)
	}

	if job.JobType != model.SyncJobTypeLuogu {
		t.Fatalf("EnqueueSync() job = %+v", job)
	}
}

func TestPlatformSyncServiceDispatchesToAtCoder(t *testing.T) {
	t.Parallel()

	service := NewPlatformSyncService(
		stubPlatformSyncAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         9,
					SiteUserID: 7,
					Platform:   model.PlatformAtCoder,
				}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(_ context.Context, siteUserID int64, accountID int64) (model.SyncJob, error) {
				if siteUserID != 7 || accountID != 9 {
					t.Fatalf("EnqueueSync() siteUserID=%d accountID=%d", siteUserID, accountID)
				}

				return model.SyncJob{ID: 2, JobType: model.SyncJobTypeAtCoder}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
				t.Fatal("codeforces enqueuer should not be called")
				return model.SyncJob{}, nil
			},
		},
		stubPlatformSyncEnqueuer{
			enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
				t.Fatal("luogu enqueuer should not be called")
				return model.SyncJob{}, nil
			},
		},
	)

	job, err := service.EnqueueSync(context.Background(), 7, 9)
	if err != nil {
		t.Fatalf("EnqueueSync() error = %v", err)
	}

	if job.JobType != model.SyncJobTypeAtCoder {
		t.Fatalf("EnqueueSync() job = %+v", job)
	}
}
