package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

type stubLuoguPlatformAccountStore struct {
	getByIDFn func(context.Context, int64) (model.PlatformAccount, error)
}

func (s stubLuoguPlatformAccountStore) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	return s.getByIDFn(ctx, accountID)
}

type stubLuoguSyncStore struct {
	saveSyncFn         func(context.Context, repository.SaveLuoguSyncParams) error
	getLatestProfileFn func(context.Context, int64) (model.PlatformProfileSnapshot, error)
	listProblemFactsFn func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error)
}

func (s stubLuoguSyncStore) SaveSync(
	ctx context.Context,
	params repository.SaveLuoguSyncParams,
) error {
	return s.saveSyncFn(ctx, params)
}

func (s stubLuoguSyncStore) GetLatestProfileSnapshot(
	ctx context.Context,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, accountID)
}

func (s stubLuoguSyncStore) ListProblemFactsByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter repository.ListPlatformSyncFilter,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, platform, filter)
}

type stubLuoguSyncJobStore struct {
	enqueueFn     func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error)
	claimNextFn   func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error)
	markSuccessFn func(context.Context, int64, time.Time) error
	markFailedFn  func(context.Context, int64, string, time.Time) error
}

func (s stubLuoguSyncJobStore) Enqueue(
	ctx context.Context,
	params repository.EnqueueSyncJobParams,
) (model.SyncJob, error) {
	return s.enqueueFn(ctx, params)
}

func (s stubLuoguSyncJobStore) ClaimNextQueuedJob(
	ctx context.Context,
	jobType model.SyncJobType,
	startedAt time.Time,
) (model.SyncJob, error) {
	return s.claimNextFn(ctx, jobType, startedAt)
}

func (s stubLuoguSyncJobStore) MarkSucceeded(
	ctx context.Context,
	jobID int64,
	finishedAt time.Time,
) error {
	return s.markSuccessFn(ctx, jobID, finishedAt)
}

func (s stubLuoguSyncJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	errorMessage string,
	finishedAt time.Time,
) error {
	return s.markFailedFn(ctx, jobID, errorMessage, finishedAt)
}

type stubLuoguSyncClient struct {
	fetchSnapshotFn func(context.Context, string) (integration.LuoguSyncSnapshot, error)
}

func (s stubLuoguSyncClient) FetchSnapshot(
	ctx context.Context,
	handle string,
) (integration.LuoguSyncSnapshot, error) {
	return s.fetchSnapshotFn(ctx, handle)
}

func TestLuoguSyncServiceSyncPersistsFetchedData(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_900_000, 0).UTC()
	service := NewLuoguSyncService(
		stubLuoguPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformLuogu,
					Handle:     "qiaochu",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubLuoguSyncStore{
			saveSyncFn: func(_ context.Context, params repository.SaveLuoguSyncParams) error {
				if params.Account.ID != 8 {
					t.Fatalf("SaveSync() account = %+v", params.Account)
				}
				if params.Profile.Source != model.SyncSourceLuoguUserInfo {
					t.Fatalf("SaveSync() profile source = %q", params.Profile.Source)
				}
				if len(params.AcceptedProblems) != 1 {
					t.Fatalf("SaveSync() accepted=%d", len(params.AcceptedProblems))
				}
				if params.AcceptedProblems[0].Source != model.SyncSourceLuoguPractice {
					t.Fatalf("SaveSync() accepted source = %q", params.AcceptedProblems[0].Source)
				}
				if !params.AcceptedProblems[0].AcceptedAt.Equal(now) {
					t.Fatalf("SaveSync() accepted_at = %v, want %v", params.AcceptedProblems[0].AcceptedAt, now)
				}
				return nil
			},
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
		},
		stubLuoguSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubLuoguSyncClient{
			fetchSnapshotFn: func(context.Context, string) (integration.LuoguSyncSnapshot, error) {
				rating := 1198
				return integration.LuoguSyncSnapshot{
					Profile: integration.LuoguProfile{
						UID:         809639,
						Handle:      "qiaochu",
						DisplayName: "qiaochu",
						Rating:      &rating,
						ProfileURL:  "https://www.luogu.com.cn/user/809639",
						Payload:     []byte(`{"uid":809639}`),
						FetchedAt:   now,
					},
					AcceptedProblems: []integration.LuoguAcceptedProblem{
						{
							UID:         809639,
							Handle:      "qiaochu",
							ProblemKey:  "P1001",
							ProblemID:   "P1001",
							ProblemName: "A+B Problem",
							ProblemURL:  "https://www.luogu.com.cn/problem/P1001",
							SourceURL:   "https://www.luogu.com.cn/user/809639/practice",
							Payload:     []byte(`{"pid":"P1001"}`),
							FetchedAt:   now,
						},
					},
				}, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	result, err := service.Sync(context.Background(), 7, 8)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.AcceptedProblemCount != 1 || result.ProblemFactCount != 1 {
		t.Fatalf("Sync() result = %+v", result)
	}
}

func TestLuoguSyncServiceEnqueueSyncUsesLuoguJobType(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_900_100, 0).UTC()
	service := NewLuoguSyncService(
		stubLuoguPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformLuogu,
					Handle:     "qiaochu",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubLuoguSyncStore{
			saveSyncFn: func(context.Context, repository.SaveLuoguSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
		},
		stubLuoguSyncJobStore{
			enqueueFn: func(_ context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				if params.JobType != model.SyncJobTypeLuogu {
					t.Fatalf("Enqueue() job type = %q, want %q", params.JobType, model.SyncJobTypeLuogu)
				}
				return model.SyncJob{ID: 1, JobType: params.JobType}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubLuoguSyncClient{
			fetchSnapshotFn: func(context.Context, string) (integration.LuoguSyncSnapshot, error) {
				return integration.LuoguSyncSnapshot{}, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	if _, err := service.EnqueueSync(context.Background(), 7, 8); err != nil {
		t.Fatalf("EnqueueSync() error = %v", err)
	}
}

func TestLuoguSyncServiceMapsUpstreamNotFound(t *testing.T) {
	t.Parallel()

	service := NewLuoguSyncService(
		stubLuoguPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformLuogu,
					Handle:     "missing",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubLuoguSyncStore{
			saveSyncFn: func(context.Context, repository.SaveLuoguSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
		},
		stubLuoguSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubLuoguSyncClient{
			fetchSnapshotFn: func(context.Context, string) (integration.LuoguSyncSnapshot, error) {
				return integration.LuoguSyncSnapshot{}, integration.ErrLuoguUserNotFound
			},
		},
	)

	_, err := service.Sync(context.Background(), 7, 8)
	var validationErr ValidationError
	if !errors.As(err, &validationErr) || validationErr.Message != "luogu handle was not found" {
		t.Fatalf("Sync() error = %v", err)
	}
}
