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

type stubAtCoderPlatformAccountStore struct {
	getByIDFn func(context.Context, int64) (model.PlatformAccount, error)
}

func (s stubAtCoderPlatformAccountStore) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	return s.getByIDFn(ctx, accountID)
}

type stubAtCoderSyncStore struct {
	saveSyncFn           func(context.Context, repository.SaveAtCoderSyncParams) error
	getLatestProfileFn   func(context.Context, int64) (model.PlatformProfileSnapshot, error)
	listContestHistoryFn func(context.Context, int64, repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error)
	listProblemFactsFn   func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ProblemFact, error)
	listContestSummaryFn func(context.Context, int64, model.Platform, repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error)
}

func (s stubAtCoderSyncStore) SaveSync(
	ctx context.Context,
	params repository.SaveAtCoderSyncParams,
) error {
	return s.saveSyncFn(ctx, params)
}

func (s stubAtCoderSyncStore) GetLatestProfileSnapshot(
	ctx context.Context,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, accountID)
}

func (s stubAtCoderSyncStore) ListContestHistoriesByAccountID(
	ctx context.Context,
	accountID int64,
	filter repository.ListPlatformSyncFilter,
) ([]model.PlatformContestHistory, error) {
	return s.listContestHistoryFn(ctx, accountID, filter)
}

func (s stubAtCoderSyncStore) ListProblemFactsByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter repository.ListPlatformSyncFilter,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, platform, filter)
}

func (s stubAtCoderSyncStore) ListContestSummariesByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter repository.ListPlatformSyncFilter,
) ([]model.ContestACSummary, error) {
	return s.listContestSummaryFn(ctx, siteUserID, platform, filter)
}

type stubAtCoderSyncJobStore struct {
	enqueueFn     func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error)
	claimNextFn   func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error)
	markSuccessFn func(context.Context, int64, time.Time) error
	markFailedFn  func(context.Context, int64, string, time.Time) error
}

func (s stubAtCoderSyncJobStore) Enqueue(
	ctx context.Context,
	params repository.EnqueueSyncJobParams,
) (model.SyncJob, error) {
	return s.enqueueFn(ctx, params)
}

func (s stubAtCoderSyncJobStore) ClaimNextQueuedJob(
	ctx context.Context,
	jobType model.SyncJobType,
	startedAt time.Time,
) (model.SyncJob, error) {
	return s.claimNextFn(ctx, jobType, startedAt)
}

func (s stubAtCoderSyncJobStore) MarkSucceeded(
	ctx context.Context,
	jobID int64,
	finishedAt time.Time,
) error {
	return s.markSuccessFn(ctx, jobID, finishedAt)
}

func (s stubAtCoderSyncJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	errorMessage string,
	finishedAt time.Time,
) error {
	return s.markFailedFn(ctx, jobID, errorMessage, finishedAt)
}

type stubAtCoderAlertStore struct {
	recordFailureFn func(context.Context, repository.RecordIntegrationAlertParams) error
	resolveFn       func(context.Context, model.IntegrationName, time.Time) error
}

func (s stubAtCoderAlertStore) RecordFailure(
	ctx context.Context,
	params repository.RecordIntegrationAlertParams,
) error {
	return s.recordFailureFn(ctx, params)
}

func (s stubAtCoderAlertStore) Resolve(
	ctx context.Context,
	integration model.IntegrationName,
	resolvedAt time.Time,
) error {
	return s.resolveFn(ctx, integration, resolvedAt)
}

type stubAtCoderSyncClient struct {
	checkReadyFn           func(context.Context) error
	fetchProfileFn         func(context.Context, string) (integration.AtCoderProfile, error)
	fetchContestHistoryFn  func(context.Context, string) ([]integration.AtCoderContestHistoryEntry, error)
	fetchAcceptedSubmitsFn func(context.Context, string, []string) ([]integration.AtCoderAcceptedSubmission, error)
}

func (s stubAtCoderSyncClient) CheckReady(ctx context.Context) error {
	return s.checkReadyFn(ctx)
}

func (s stubAtCoderSyncClient) FetchProfile(
	ctx context.Context,
	handle string,
) (integration.AtCoderProfile, error) {
	return s.fetchProfileFn(ctx, handle)
}

func (s stubAtCoderSyncClient) FetchContestHistory(
	ctx context.Context,
	handle string,
) ([]integration.AtCoderContestHistoryEntry, error) {
	return s.fetchContestHistoryFn(ctx, handle)
}

func (s stubAtCoderSyncClient) FetchAcceptedSubmissions(
	ctx context.Context,
	handle string,
	contestIDs []string,
) ([]integration.AtCoderAcceptedSubmission, error) {
	return s.fetchAcceptedSubmitsFn(ctx, handle, contestIDs)
}

func TestAtCoderSyncServiceSyncPersistsFetchedDataAndResolvesAlert(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_920_000, 0).UTC()
	resolved := false

	service := NewAtCoderSyncService(
		stubAtCoderPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformAtCoder,
					Handle:     "tourist",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubAtCoderSyncStore{
			saveSyncFn: func(_ context.Context, params repository.SaveAtCoderSyncParams) error {
				if params.Account.ID != 8 {
					t.Fatalf("SaveSync() account = %+v", params.Account)
				}
				if params.Profile.Source != model.SyncSourceAtCoderProfilePage {
					t.Fatalf("SaveSync() profile source = %q", params.Profile.Source)
				}
				if len(params.AcceptedEvents) != 2 || len(params.ContestHistories) != 2 {
					t.Fatalf("SaveSync() accepted=%d contests=%d", len(params.AcceptedEvents), len(params.ContestHistories))
				}
				if params.AcceptedEvents[0].Source != model.SyncSourceAtCoderSubmission {
					t.Fatalf("SaveSync() accepted source = %q", params.AcceptedEvents[0].Source)
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
		stubAtCoderSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubAtCoderAlertStore{
			recordFailureFn: func(context.Context, repository.RecordIntegrationAlertParams) error {
				t.Fatal("RecordFailure() should not be called on successful sync")
				return nil
			},
			resolveFn: func(_ context.Context, integrationName model.IntegrationName, resolvedAt time.Time) error {
				if integrationName != model.IntegrationAtCoderMain {
					t.Fatalf("Resolve() integration = %q", integrationName)
				}
				if !resolvedAt.Equal(now) {
					t.Fatalf("Resolve() resolvedAt = %v, want %v", resolvedAt, now)
				}
				resolved = true
				return nil
			},
		},
		stubAtCoderSyncClient{
			checkReadyFn: func(context.Context) error { return nil },
			fetchProfileFn: func(context.Context, string) (integration.AtCoderProfile, error) {
				rating := 3797
				maxRating := 4229
				return integration.AtCoderProfile{
					Handle:      "tourist",
					DisplayName: "tourist",
					Rating:      &rating,
					MaxRating:   &maxRating,
					ProfileURL:  "https://atcoder.jp/users/tourist",
					Payload:     []byte(`{"profile":true}`),
					FetchedAt:   now,
				}, nil
			},
			fetchContestHistoryFn: func(context.Context, string) ([]integration.AtCoderContestHistoryEntry, error) {
				rank := 1
				oldRating := 3790
				newRating := 3797
				delta := 7
				return []integration.AtCoderContestHistoryEntry{
					{
						ContestID:      "agc077",
						ContestName:    "AtCoder Grand Contest 077",
						Rank:           &rank,
						OldRating:      &oldRating,
						NewRating:      &newRating,
						RatingDelta:    &delta,
						ParticipatedAt: now.Add(-2 * time.Hour),
						SourceURL:      "https://atcoder.jp/contests/agc077",
						Payload:        []byte(`{"contest":"agc077"}`),
						FetchedAt:      now,
					},
					{
						ContestID:      "abc999",
						ContestName:    "AtCoder Beginner Contest 999",
						ParticipatedAt: now.Add(-time.Hour),
						SourceURL:      "https://atcoder.jp/contests/abc999",
						Payload:        []byte(`{"contest":"abc999"}`),
						FetchedAt:      now,
					},
				}, nil
			},
			fetchAcceptedSubmitsFn: func(_ context.Context, handle string, contestIDs []string) ([]integration.AtCoderAcceptedSubmission, error) {
				if handle != "tourist" {
					t.Fatalf("FetchAcceptedSubmissions() handle = %q", handle)
				}
				if len(contestIDs) != 2 {
					t.Fatalf("FetchAcceptedSubmissions() contestIDs = %#v", contestIDs)
				}
				return []integration.AtCoderAcceptedSubmission{
					{
						Handle:       "tourist",
						ProblemKey:   "agc077_a",
						ContestID:    "agc077",
						TaskID:       "agc077_a",
						ProblemName:  "A - Candy",
						ProblemURL:   "https://atcoder.jp/contests/agc077/tasks/agc077_a",
						AcceptedAt:   now.Add(-30 * time.Minute),
						SubmissionID: "6001",
						SourceURL:    "https://atcoder.jp/contests/agc077/submissions/6001",
						Payload:      []byte(`{"submission":"6001"}`),
						FetchedAt:    now,
					},
					{
						Handle:       "tourist",
						ProblemKey:   "abc999_b",
						ContestID:    "abc999",
						TaskID:       "abc999_b",
						ProblemName:  "B - Demo",
						ProblemURL:   "https://atcoder.jp/contests/abc999/tasks/abc999_b",
						AcceptedAt:   now.Add(-20 * time.Minute),
						SubmissionID: "6002",
						SourceURL:    "https://atcoder.jp/contests/abc999/submissions/6002",
						Payload:      []byte(`{"submission":"6002"}`),
						FetchedAt:    now,
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

	if !resolved {
		t.Fatal("Sync() expected Resolve() to be called")
	}
	if result.AcceptedEventCount != 2 || result.ProblemFactCount != 2 || result.ContestSummaryCount != 2 || result.ContestHistoryCount != 2 {
		t.Fatalf("Sync() result = %+v", result)
	}
}

func TestAtCoderSyncServiceEnqueueSyncRecordsAlertWhenCredentialIsMissing(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_920_100, 0).UTC()
	recorded := false

	service := NewAtCoderSyncService(
		stubAtCoderPlatformAccountStore{
			getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
				return model.PlatformAccount{
					ID:         8,
					SiteUserID: 7,
					Platform:   model.PlatformAtCoder,
					Handle:     "tourist",
					Status:     model.PlatformAccountStatusVerified,
				}, nil
			},
		},
		stubAtCoderSyncStore{
			saveSyncFn: func(context.Context, repository.SaveAtCoderSyncParams) error { return nil },
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
		stubAtCoderSyncJobStore{
			enqueueFn: func(context.Context, repository.EnqueueSyncJobParams) (model.SyncJob, error) {
				t.Fatal("Enqueue() should not be called when preflight fails")
				return model.SyncJob{}, nil
			},
			claimNextFn: func(context.Context, model.SyncJobType, time.Time) (model.SyncJob, error) {
				return model.SyncJob{}, repository.ErrNoPendingSyncJob
			},
			markSuccessFn: func(context.Context, int64, time.Time) error { return nil },
			markFailedFn:  func(context.Context, int64, string, time.Time) error { return nil },
		},
		stubAtCoderAlertStore{
			recordFailureFn: func(_ context.Context, params repository.RecordIntegrationAlertParams) error {
				if params.Integration != model.IntegrationAtCoderMain {
					t.Fatalf("RecordFailure() integration = %q", params.Integration)
				}
				recorded = true
				return nil
			},
			resolveFn: func(context.Context, model.IntegrationName, time.Time) error {
				t.Fatal("Resolve() should not be called on preflight failure")
				return nil
			},
		},
		stubAtCoderSyncClient{
			checkReadyFn: func(context.Context) error {
				return integration.ErrAtCoderCredentialMissing
			},
			fetchProfileFn: func(context.Context, string) (integration.AtCoderProfile, error) {
				return integration.AtCoderProfile{}, nil
			},
			fetchContestHistoryFn: func(context.Context, string) ([]integration.AtCoderContestHistoryEntry, error) {
				return nil, nil
			},
			fetchAcceptedSubmitsFn: func(context.Context, string, []string) ([]integration.AtCoderAcceptedSubmission, error) {
				return nil, nil
			},
		},
	)
	service.now = func() time.Time { return now }

	_, err := service.EnqueueSync(context.Background(), 7, 8)
	if !errors.Is(err, ErrAtCoderUpstream) {
		t.Fatalf("EnqueueSync() error = %v, want %v", err, ErrAtCoderUpstream)
	}
	if !recorded {
		t.Fatal("EnqueueSync() expected RecordFailure() to be called")
	}
}
