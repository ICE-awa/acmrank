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

type stubICPCAwardUserStore struct {
	getByIDFn func(context.Context, int64) (model.User, error)
}

func (s stubICPCAwardUserStore) GetByID(
	ctx context.Context,
	id int64,
) (model.User, error) {
	return s.getByIDFn(ctx, id)
}

type stubICPCAwardStore struct {
	replaceFn func(context.Context, repository.ReplaceAwardRecordsParams) error
	listFn    func(context.Context, int64, repository.ListAwardRecordsFilter) ([]model.AwardRecord, error)
}

func (s stubICPCAwardStore) ReplaceAutoSyncByUserID(
	ctx context.Context,
	params repository.ReplaceAwardRecordsParams,
) error {
	return s.replaceFn(ctx, params)
}

func (s stubICPCAwardStore) ListByUserID(
	ctx context.Context,
	siteUserID int64,
	filter repository.ListAwardRecordsFilter,
) ([]model.AwardRecord, error) {
	return s.listFn(ctx, siteUserID, filter)
}

type stubICPCAwardJobStore struct {
	enqueueFn     func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error)
	claimNextFn   func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error)
	markSuccessFn func(context.Context, int64, time.Time) error
	markFailedFn  func(context.Context, int64, string, time.Time) error
}

func (s stubICPCAwardJobStore) Enqueue(
	ctx context.Context,
	params repository.EnqueueSyncJobParams,
) (model.SyncJob, error) {
	return s.enqueueFn(ctx, params)
}

func (s stubICPCAwardJobStore) ClaimNextQueuedJob(
	ctx context.Context,
	jobType model.SyncJobType,
	startedAt time.Time,
) (model.SyncJob, error) {
	return s.claimNextFn(ctx, jobType, startedAt)
}

func (s stubICPCAwardJobStore) MarkSucceeded(
	ctx context.Context,
	jobID int64,
	finishedAt time.Time,
) error {
	return s.markSuccessFn(ctx, jobID, finishedAt)
}

func (s stubICPCAwardJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	errorMessage string,
	finishedAt time.Time,
) error {
	return s.markFailedFn(ctx, jobID, errorMessage, finishedAt)
}

type stubICPCAwardClient struct {
	fetchAwardsFn func(context.Context) ([]integration.ICPCAwardFeedRecord, error)
}

func (s stubICPCAwardClient) FetchAwards(
	ctx context.Context,
) ([]integration.ICPCAwardFeedRecord, error) {
	return s.fetchAwardsFn(ctx)
}

func TestICPCAwardServiceSyncMatchesRealNameAndReplacesAwards(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_701_000_000, 0).UTC()
	service := NewICPCAwardService(
		stubICPCAwardUserStore{
			getByIDFn: func(context.Context, int64) (model.User, error) {
				return model.User{
					ID:       7,
					Username: "tourist",
					RealName: "Alice Zhang",
					Status:   model.UserStatusActive,
				}, nil
			},
		},
		stubICPCAwardStore{
			replaceFn: func(_ context.Context, params repository.ReplaceAwardRecordsParams) error {
				if params.SiteUserID != 7 || params.Platform != model.AwardPlatformICPC {
					t.Fatalf("ReplaceAutoSyncByUserID() params = %+v", params)
				}
				if len(params.Records) != 1 {
					t.Fatalf("ReplaceAutoSyncByUserID() records = %#v", params.Records)
				}
				if params.Records[0].ContestName != "ICPC Asia Regional 2025" {
					t.Fatalf("ContestName = %q", params.Records[0].ContestName)
				}
				return nil
			},
			listFn: func(context.Context, int64, repository.ListAwardRecordsFilter) ([]model.AwardRecord, error) {
				return nil, nil
			},
		},
		stubICPCAwardJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubICPCAwardClient{
			fetchAwardsFn: func(context.Context) ([]integration.ICPCAwardFeedRecord, error) {
				return []integration.ICPCAwardFeedRecord{
					{
						ContestName: "ICPC Asia Regional 2025",
						AwardName:   "Gold Medal",
						RankText:    "Rank 3",
						AwardDate:   time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC),
						Members:     []string{"Alice  Zhang", "Bob Li"},
						NormalizedMembers: []string{
							"alicezhang",
							"bobli",
						},
						SourceURL: "https://board.example.test/regional-2025",
					},
					{
						ContestName: "ICPC Asia Regional 2025",
						AwardName:   "Gold Medal",
						RankText:    "Rank 3",
						AwardDate:   time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC),
						Members:     []string{"Alice Zhang"},
						NormalizedMembers: []string{
							"alicezhang",
						},
						SourceURL: "https://board.example.test/regional-2025",
					},
					{
						ContestName: "ICPC EC Final 2024",
						AwardName:   "Silver Medal",
						AwardDate:   time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC),
						Members:     []string{"Carol"},
						NormalizedMembers: []string{
							"carol",
						},
					},
				}, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	result, err := service.Sync(context.Background(), 7)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.MatchedAwardCount != 1 {
		t.Fatalf("MatchedAwardCount = %d, want 1", result.MatchedAwardCount)
	}
}

func TestICPCAwardServiceEnqueueSyncUsesUserLevelJob(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_701_000_100, 0).UTC()
	service := NewICPCAwardService(
		stubICPCAwardUserStore{
			getByIDFn: func(context.Context, int64) (model.User, error) {
				return model.User{ID: 7, RealName: "Alice", Status: model.UserStatusActive}, nil
			},
		},
		stubICPCAwardStore{
			replaceFn: func(context.Context, repository.ReplaceAwardRecordsParams) error { return nil },
			listFn: func(context.Context, int64, repository.ListAwardRecordsFilter) ([]model.AwardRecord, error) {
				return nil, nil
			},
		},
		stubICPCAwardJobStore{
			enqueueFn: func(_ context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				if params.JobType != model.SyncJobTypeICPCAward {
					t.Fatalf("JobType = %q, want %q", params.JobType, model.SyncJobTypeICPCAward)
				}
				if params.PlatformAccountID != nil {
					t.Fatalf("PlatformAccountID = %#v, want nil", params.PlatformAccountID)
				}
				return model.SyncJob{ID: 9, JobType: params.JobType}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubICPCAwardClient{
			fetchAwardsFn: func(context.Context) ([]integration.ICPCAwardFeedRecord, error) {
				return nil, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	if _, err := service.EnqueueSync(context.Background(), 7); err != nil {
		t.Fatalf("EnqueueSync() error = %v", err)
	}
}

func TestICPCAwardServiceProcessNextQueuedSyncClaimsICPCJobs(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_701_000_200, 0).UTC()
	siteUserID := int64(7)
	markedSucceeded := false

	service := NewICPCAwardService(
		stubICPCAwardUserStore{
			getByIDFn: func(context.Context, int64) (model.User, error) {
				return model.User{
					ID:       7,
					RealName: "Alice Zhang",
					Status:   model.UserStatusActive,
				}, nil
			},
		},
		stubICPCAwardStore{
			replaceFn: func(context.Context, repository.ReplaceAwardRecordsParams) error { return nil },
			listFn: func(context.Context, int64, repository.ListAwardRecordsFilter) ([]model.AwardRecord, error) {
				return nil, nil
			},
		},
		stubICPCAwardJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(_ context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error) {
				if jobType != model.SyncJobTypeICPCAward {
					t.Fatalf("ClaimNextQueuedJob() jobType = %q, want %q", jobType, model.SyncJobTypeICPCAward)
				}
				if !startedAt.Equal(now) {
					t.Fatalf("startedAt = %v, want %v", startedAt, now)
				}
				return model.SyncJob{
					ID:         12,
					SiteUserID: &siteUserID,
					Platform:   model.AwardPlatformICPC,
					JobType:    model.SyncJobTypeICPCAward,
					Status:     model.SyncJobStatusRunning,
				}, nil
			},
			markSuccessFn: func(context.Context, int64, time.Time) error {
				markedSucceeded = true
				return nil
			},
			markFailedFn: func(context.Context, int64, string, time.Time) error {
				return errors.New("unexpected failure")
			},
		},
		stubICPCAwardClient{
			fetchAwardsFn: func(context.Context) ([]integration.ICPCAwardFeedRecord, error) {
				return []integration.ICPCAwardFeedRecord{
					{
						ContestName: "ICPC Asia Regional 2025",
						AwardName:   "Gold Medal",
						AwardDate:   time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC),
						Members:     []string{"Alice Zhang"},
						NormalizedMembers: []string{
							"alicezhang",
						},
					},
				}, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	processed, err := service.ProcessNextQueuedSync(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedSync() error = %v", err)
	}

	if !processed || !markedSucceeded {
		t.Fatalf("processed=%v markedSucceeded=%v, want true/true", processed, markedSucceeded)
	}
}

func TestAwardRecordMatchesRealNameUsesPreNormalizedMembers(t *testing.T) {
	t.Parallel()

	matched := awardRecordMatchesRealName(
		"alicezhang",
		integration.ICPCAwardFeedRecord{
			Members: []string{"not-a-match"},
			NormalizedMembers: []string{
				"alicezhang",
			},
		},
	)
	if !matched {
		t.Fatal("awardRecordMatchesRealName() = false, want true")
	}
}
