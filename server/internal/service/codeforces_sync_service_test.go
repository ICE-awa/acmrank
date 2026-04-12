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

type stubCodeforcesPlatformAccountStore struct {
	getByIDFn func(context.Context, int64) (model.PlatformAccount, error)
}

func (s stubCodeforcesPlatformAccountStore) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	return s.getByIDFn(ctx, accountID)
}

type stubCodeforcesSyncStore struct {
	saveSyncFn           func(context.Context, repository.SaveCodeforcesSyncParams) error
	getLatestProfileFn   func(context.Context, int64) (model.PlatformProfileSnapshot, error)
	listContestHistoryFn func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error)
	listProblemFactsFn   func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error)
	listContestSummaryFn func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error)
}

func (s stubCodeforcesSyncStore) SaveSync(
	ctx context.Context,
	params repository.SaveCodeforcesSyncParams,
) error {
	return s.saveSyncFn(ctx, params)
}

func (s stubCodeforcesSyncStore) GetLatestProfileSnapshot(
	ctx context.Context,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, accountID)
}

func (s stubCodeforcesSyncStore) ListContestHistoriesByAccountID(
	ctx context.Context,
	accountID int64,
	filter repository.ListPlatformSyncFilter,
) ([]model.PlatformContestHistory, error) {
	return s.listContestHistoryFn(ctx, accountID, filter)
}

func (s stubCodeforcesSyncStore) ListProblemFactsByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter repository.ListPlatformSyncFilter,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, platform, filter)
}

func (s stubCodeforcesSyncStore) ListContestSummariesByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter repository.ListPlatformSyncFilter,
) ([]model.ContestACSummary, error) {
	return s.listContestSummaryFn(ctx, siteUserID, platform, filter)
}

type stubCodeforcesSyncJobStore struct {
	enqueueFn     func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error)
	claimNextFn   func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error)
	markSuccessFn func(context.Context, int64, time.Time) error
	markFailedFn  func(context.Context, int64, string, time.Time) error
}

func (s stubCodeforcesSyncJobStore) Enqueue(
	ctx context.Context,
	params repository.EnqueueSyncJobParams,
) (model.SyncJob, error) {
	return s.enqueueFn(ctx, params)
}

func (s stubCodeforcesSyncJobStore) ClaimNextQueuedJob(
	ctx context.Context,
	jobType model.SyncJobType,
	startedAt time.Time,
) (model.SyncJob, error) {
	return s.claimNextFn(ctx, jobType, startedAt)
}

func (s stubCodeforcesSyncJobStore) MarkSucceeded(
	ctx context.Context,
	jobID int64,
	finishedAt time.Time,
) error {
	return s.markSuccessFn(ctx, jobID, finishedAt)
}

func (s stubCodeforcesSyncJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	errorMessage string,
	finishedAt time.Time,
) error {
	return s.markFailedFn(ctx, jobID, errorMessage, finishedAt)
}

type stubCodeforcesSyncClient struct {
	fetchProfileFn             func(context.Context, string) (integration.CodeforcesProfile, error)
	fetchAcceptedSubmissionsFn func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error)
	fetchContestHistoryFn      func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error)
}

func (s stubCodeforcesSyncClient) FetchProfile(
	ctx context.Context,
	handle string,
) (integration.CodeforcesProfile, error) {
	return s.fetchProfileFn(ctx, handle)
}

func (s stubCodeforcesSyncClient) FetchAcceptedSubmissions(
	ctx context.Context,
	handle string,
) ([]integration.CodeforcesAcceptedSubmission, error) {
	return s.fetchAcceptedSubmissionsFn(ctx, handle)
}

func (s stubCodeforcesSyncClient) FetchContestHistory(
	ctx context.Context,
	handle string,
) ([]integration.CodeforcesContestHistoryEntry, error) {
	return s.fetchContestHistoryFn(ctx, handle)
}

func TestCodeforcesSyncServiceSyncPersistsFetchedData(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_400_000, 0).UTC()
	service := NewCodeforcesSyncService(
		stubCodeforcesPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformCodeforces,
					Handle:     "tourist",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubCodeforcesSyncStore{
			saveSyncFn: func(_ context.Context, params repository.SaveCodeforcesSyncParams) error {
				if params.Account.ID != 8 {
					t.Fatalf("SaveSync() account = %+v", params.Account)
				}
				if params.Profile.Source != model.SyncSourceCodeforcesAPI {
					t.Fatalf("SaveSync() profile source = %q", params.Profile.Source)
				}
				if len(params.AcceptedEvents) != 2 || len(params.ContestHistories) != 1 {
					t.Fatalf("SaveSync() accepted=%d contests=%d", len(params.AcceptedEvents), len(params.ContestHistories))
				}
				if params.AcceptedEvents[0].Source != model.SyncSourceCodeforcesAPI {
					t.Fatalf("SaveSync() accepted event source = %q", params.AcceptedEvents[0].Source)
				}
				return nil
			},
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listContestHistoryFn: func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error) {
				return nil, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
			listContestSummaryFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error) {
				return nil, nil
			},
		},
		stubCodeforcesSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubCodeforcesSyncClient{
			fetchProfileFn: func(context.Context, string) (integration.CodeforcesProfile, error) {
				rating := 3500
				maxRating := 3826
				return integration.CodeforcesProfile{
					Handle:      "tourist",
					DisplayName: "tourist",
					Rating:      &rating,
					MaxRating:   &maxRating,
					ProfileURL:  "https://codeforces.com/profile/tourist",
					Payload:     []byte(`{"handle":"tourist"}`),
					FetchedAt:   now,
				}, nil
			},
			fetchAcceptedSubmissionsFn: func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error) {
				return []integration.CodeforcesAcceptedSubmission{
					{
						Handle:       "tourist",
						ProblemKey:   "CF-1000A",
						ContestID:    "1000",
						ProblemIndex: "A",
						ProblemName:  "Problem A",
						ProblemURL:   "https://codeforces.com/contest/1000/problem/A",
						AcceptedAt:   now.Add(-time.Hour),
						SubmissionID: "1",
						SourceURL:    "https://codeforces.com/contest/1000/submission/1",
						Payload:      []byte(`{"id":1}`),
						FetchedAt:    now,
					},
					{
						Handle:       "tourist",
						ProblemKey:   "CF-1000B",
						ContestID:    "1000",
						ProblemIndex: "B",
						ProblemName:  "Problem B",
						ProblemURL:   "https://codeforces.com/contest/1000/problem/B",
						AcceptedAt:   now,
						SubmissionID: "2",
						SourceURL:    "https://codeforces.com/contest/1000/submission/2",
						Payload:      []byte(`{"id":2}`),
						FetchedAt:    now,
					},
				}, nil
			},
			fetchContestHistoryFn: func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error) {
				return []integration.CodeforcesContestHistoryEntry{
					{
						ContestID:      "1000",
						ContestName:    "Round 1000",
						Rank:           5,
						OldRating:      3400,
						NewRating:      3500,
						RatingDelta:    100,
						ParticipatedAt: now.Add(-2 * time.Hour),
						SourceURL:      "https://codeforces.com/contest/1000",
						Payload:        []byte(`{"contestId":1000}`),
						FetchedAt:      now,
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

	if result.AcceptedEventCount != 2 || result.ProblemFactCount != 2 || result.ContestSummaryCount != 1 || result.ContestHistoryCount != 1 {
		t.Fatalf("Sync() result = %+v", result)
	}
}

func TestCodeforcesSyncServiceSyncRejectsUnverifiedAccount(t *testing.T) {
	t.Parallel()

	service := NewCodeforcesSyncService(
		stubCodeforcesPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformCodeforces,
					Status:     model.PlatformAccountStatusPendingReview,
				}, nil
			},
		},
		stubCodeforcesSyncStore{
			saveSyncFn: func(context.Context, repository.SaveCodeforcesSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listContestHistoryFn: func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error) {
				return nil, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
			listContestSummaryFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error) {
				return nil, nil
			},
		},
		stubCodeforcesSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubCodeforcesSyncClient{
			fetchProfileFn: func(context.Context, string) (integration.CodeforcesProfile, error) {
				return integration.CodeforcesProfile{}, nil
			},
			fetchAcceptedSubmissionsFn: func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error) {
				return nil, nil
			},
			fetchContestHistoryFn: func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error) {
				return nil, nil
			},
		},
	)

	_, err := service.Sync(context.Background(), 7, 8)
	if !errors.Is(err, ErrPlatformAccountNotReady) {
		t.Fatalf("Sync() error = %v, want %v", err, ErrPlatformAccountNotReady)
	}
}

func TestCodeforcesSyncServiceGetLatestProfileMapsNotFound(t *testing.T) {
	t.Parallel()

	service := NewCodeforcesSyncService(
		stubCodeforcesPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformCodeforces,
				}, nil
			},
		},
		stubCodeforcesSyncStore{
			saveSyncFn: func(context.Context, repository.SaveCodeforcesSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, repository.ErrPlatformSyncDataNotFound
			},
			listContestHistoryFn: func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error) {
				return nil, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
			listContestSummaryFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error) {
				return nil, nil
			},
		},
		stubCodeforcesSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubCodeforcesSyncClient{
			fetchProfileFn: func(context.Context, string) (integration.CodeforcesProfile, error) {
				return integration.CodeforcesProfile{}, nil
			},
			fetchAcceptedSubmissionsFn: func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error) {
				return nil, nil
			},
			fetchContestHistoryFn: func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error) {
				return nil, nil
			},
		},
	)

	_, err := service.GetLatestProfile(context.Background(), 7, 8)
	if !errors.Is(err, ErrCodeforcesSyncDataNotFound) {
		t.Fatalf("GetLatestProfile() error = %v, want %v", err, ErrCodeforcesSyncDataNotFound)
	}
}

func TestCodeforcesSyncServiceEnqueueSyncCreatesQueuedJob(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_400_500, 0).UTC()
	service := NewCodeforcesSyncService(
		stubCodeforcesPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformCodeforces,
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubCodeforcesSyncStore{
			saveSyncFn: func(context.Context, repository.SaveCodeforcesSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listContestHistoryFn: func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error) {
				return nil, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
			listContestSummaryFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error) {
				return nil, nil
			},
		},
		stubCodeforcesSyncJobStore{
			enqueueFn: func(_ context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				if params.SiteUserID != 7 || params.PlatformAccountID == nil || *params.PlatformAccountID != 8 || params.JobType != model.SyncJobTypeCodeforces {
					t.Fatalf("Enqueue() params = %+v", params)
				}

				accountID := int64(8)
				return model.SyncJob{
					ID:                11,
					PlatformAccountID: &accountID,
					Platform:          "codeforces",
					JobType:           model.SyncJobTypeCodeforces,
					Status:            model.SyncJobStatusQueued,
					ScheduledAt:       now,
					CreatedAt:         now,
					UpdatedAt:         now,
				}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubCodeforcesSyncClient{
			fetchProfileFn: func(context.Context, string) (integration.CodeforcesProfile, error) {
				return integration.CodeforcesProfile{}, nil
			},
			fetchAcceptedSubmissionsFn: func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error) { return nil, nil },
			fetchContestHistoryFn:      func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error) { return nil, nil },
		},
	)
	service.now = func() time.Time { return now }

	job, err := service.EnqueueSync(context.Background(), 7, 8)
	if err != nil {
		t.Fatalf("EnqueueSync() error = %v", err)
	}
	if job.ID != 11 || job.Status != model.SyncJobStatusQueued {
		t.Fatalf("EnqueueSync() job = %+v", job)
	}
}

func TestCodeforcesSyncServiceProcessNextQueuedSyncRunsAndMarksSuccess(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_400_700, 0).UTC()
	accountID := int64(8)
	siteUserID := int64(7)
	markSucceededCalled := false
	service := NewCodeforcesSyncService(
		stubCodeforcesPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformCodeforces,
					Handle:     "tourist",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubCodeforcesSyncStore{
			saveSyncFn: func(context.Context, repository.SaveCodeforcesSyncParams) error { return nil },
			getLatestProfileFn: func(context.Context, int64) (model.PlatformProfileSnapshot, error) {
				return model.PlatformProfileSnapshot{}, nil
			},
			listContestHistoryFn: func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error) {
				return nil, nil
			},
			listProblemFactsFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error) {
				return nil, nil
			},
			listContestSummaryFn: func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error) {
				return nil, nil
			},
		},
		stubCodeforcesSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(_ context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error) {
				if jobType != model.SyncJobTypeCodeforces {
					t.Fatalf("ClaimNextQueuedJob() jobType = %q", jobType)
				}

				return model.SyncJob{
					ID:                21,
					SiteUserID:        &siteUserID,
					PlatformAccountID: &accountID,
					Platform:          "codeforces",
					JobType:           model.SyncJobTypeCodeforces,
					Status:            model.SyncJobStatusRunning,
					ScheduledAt:       startedAt,
				}, nil
			},
			markSuccessFn: func(context.Context, int64, time.Time) error {
				markSucceededCalled = true
				return nil
			},
			markFailedFn: func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubCodeforcesSyncClient{
			fetchProfileFn: func(context.Context, string) (integration.CodeforcesProfile, error) {
				return integration.CodeforcesProfile{DisplayName: "tourist", Payload: []byte(`{}`), FetchedAt: now}, nil
			},
			fetchAcceptedSubmissionsFn: func(context.Context, string) ([]integration.CodeforcesAcceptedSubmission, error) { return nil, nil },
			fetchContestHistoryFn:      func(context.Context, string) ([]integration.CodeforcesContestHistoryEntry, error) { return nil, nil },
		},
	)
	service.now = func() time.Time { return now }

	processed, err := service.ProcessNextQueuedSync(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedSync() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNextQueuedSync() processed = false, want true")
	}
	if !markSucceededCalled {
		t.Fatal("ProcessNextQueuedSync() expected MarkSucceeded to be called")
	}
}
