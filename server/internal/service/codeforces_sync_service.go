package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
	"golang.org/x/sync/errgroup"
)

var (
	ErrPlatformAccountNotReady    = errors.New("platform account must be verified before sync")
	ErrCodeforcesUpstream         = errors.New("codeforces upstream unavailable")
	ErrCodeforcesSyncDataNotFound = errors.New("codeforces sync data not found")
)

const (
	defaultCodeforcesListLimit = 50
	maxCodeforcesListLimit     = 100
)

type CodeforcesPlatformAccountStore interface {
	GetByID(ctx context.Context, accountID int64) (model.PlatformAccount, error)
}

type CodeforcesSyncStore interface {
	SaveSync(ctx context.Context, params repository.SaveCodeforcesSyncParams) error
	GetLatestProfileSnapshot(ctx context.Context, accountID int64) (model.PlatformProfileSnapshot, error)
	ListContestHistoriesByAccountID(ctx context.Context, accountID int64, filter repository.ListCodeforcesSyncFilter) ([]model.PlatformContestHistory, error)
	ListProblemFactsByUserIDAndPlatform(ctx context.Context, siteUserID int64, platform model.Platform, filter repository.ListCodeforcesSyncFilter) ([]model.ProblemFact, error)
	ListContestSummariesByUserIDAndPlatform(ctx context.Context, siteUserID int64, platform model.Platform, filter repository.ListCodeforcesSyncFilter) ([]model.ContestACSummary, error)
}

type CodeforcesSyncJobStore interface {
	Enqueue(ctx context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error)
	ClaimNextQueuedJob(ctx context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error)
	MarkSucceeded(ctx context.Context, jobID int64, finishedAt time.Time) error
	MarkFailed(ctx context.Context, jobID int64, errorMessage string, finishedAt time.Time) error
}

type CodeforcesSyncClient interface {
	FetchProfile(ctx context.Context, handle string) (integration.CodeforcesProfile, error)
	FetchAcceptedSubmissions(ctx context.Context, handle string) ([]integration.CodeforcesAcceptedSubmission, error)
	FetchContestHistory(ctx context.Context, handle string) ([]integration.CodeforcesContestHistoryEntry, error)
}

type ListCodeforcesSyncInput struct {
	Limit  int
	Offset int
}

type CodeforcesSyncResult struct {
	SyncedAt            time.Time
	AcceptedEventCount  int
	ProblemFactCount    int
	ContestSummaryCount int
	ContestHistoryCount int
}

type CodeforcesSyncService struct {
	accountStore CodeforcesPlatformAccountStore
	syncStore    CodeforcesSyncStore
	jobStore     CodeforcesSyncJobStore
	client       CodeforcesSyncClient
	now          func() time.Time
}

func NewCodeforcesSyncService(
	accountStore CodeforcesPlatformAccountStore,
	syncStore CodeforcesSyncStore,
	jobStore CodeforcesSyncJobStore,
	client CodeforcesSyncClient,
) *CodeforcesSyncService {
	return &CodeforcesSyncService{
		accountStore: accountStore,
		syncStore:    syncStore,
		jobStore:     jobStore,
		client:       client,
		now:          time.Now,
	}
}

func (s *CodeforcesSyncService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	account, err := s.loadOwnedCodeforcesAccount(ctx, siteUserID, accountID)
	if err != nil {
		return model.SyncJob{}, err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return model.SyncJob{}, ErrPlatformAccountNotReady
	}

	job, err := s.jobStore.Enqueue(ctx, repository.EnqueueSyncJobParams{
		SiteUserID:        siteUserID,
		PlatformAccountID: account.ID,
		Platform:          string(account.Platform),
		JobType:           model.SyncJobTypeCodeforces,
		ScheduledAt:       s.now().UTC(),
	})
	if err != nil {
		return model.SyncJob{}, fmt.Errorf("enqueue codeforces sync: %w", err)
	}

	return job, nil
}

func (s *CodeforcesSyncService) ProcessNextQueuedSync(
	ctx context.Context,
) (bool, error) {
	startedAt := s.now().UTC()
	job, err := s.jobStore.ClaimNextQueuedJob(ctx, model.SyncJobTypeCodeforces, startedAt)
	if err != nil {
		if errors.Is(err, repository.ErrNoPendingSyncJob) {
			return false, nil
		}

		return false, fmt.Errorf("claim codeforces sync job: %w", err)
	}

	processErr := s.processSyncJob(ctx, job)
	finishedAt := s.now().UTC()
	if processErr != nil {
		if markErr := s.jobStore.MarkFailed(ctx, job.ID, syncJobErrorMessage(processErr), finishedAt); markErr != nil {
			return true, errors.Join(processErr, fmt.Errorf("mark codeforces sync job failed: %w", markErr))
		}

		return true, fmt.Errorf("process codeforces sync job %d: %w", job.ID, processErr)
	}

	if err := s.jobStore.MarkSucceeded(ctx, job.ID, finishedAt); err != nil {
		return true, fmt.Errorf("mark codeforces sync job succeeded: %w", err)
	}

	return true, nil
}

func (s *CodeforcesSyncService) Sync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (CodeforcesSyncResult, error) {
	account, err := s.loadOwnedCodeforcesAccount(ctx, siteUserID, accountID)
	if err != nil {
		return CodeforcesSyncResult{}, err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return CodeforcesSyncResult{}, ErrPlatformAccountNotReady
	}

	return s.syncAccount(ctx, account)
}

func (s *CodeforcesSyncService) processSyncJob(
	ctx context.Context,
	job model.SyncJob,
) error {
	if job.PlatformAccountID == nil {
		return errors.New("sync job is missing platform account id")
	}
	if job.SiteUserID == nil {
		return errors.New("sync job is missing site user id")
	}

	account, err := s.loadOwnedCodeforcesAccount(ctx, *job.SiteUserID, *job.PlatformAccountID)
	if err != nil {
		return err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return ErrPlatformAccountNotReady
	}

	_, err = s.syncAccount(ctx, account)
	return err
}

func (s *CodeforcesSyncService) syncAccount(
	ctx context.Context,
	account model.PlatformAccount,
) (CodeforcesSyncResult, error) {

	var profile integration.CodeforcesProfile
	var acceptedSubmissions []integration.CodeforcesAcceptedSubmission
	var contestHistory []integration.CodeforcesContestHistoryEntry

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var fetchErr error
		profile, fetchErr = s.client.FetchProfile(groupCtx, account.Handle)
		return fetchErr
	})
	group.Go(func() error {
		var fetchErr error
		acceptedSubmissions, fetchErr = s.client.FetchAcceptedSubmissions(groupCtx, account.Handle)
		return fetchErr
	})
	group.Go(func() error {
		var fetchErr error
		contestHistory, fetchErr = s.client.FetchContestHistory(groupCtx, account.Handle)
		return fetchErr
	})

	if err := group.Wait(); err != nil {
		return CodeforcesSyncResult{}, mapCodeforcesClientError(err)
	}

	syncedAt := s.now().UTC()
	if err := s.syncStore.SaveSync(ctx, repository.SaveCodeforcesSyncParams{
		Account:  account,
		SyncedAt: syncedAt,
		Profile: repository.CodeforcesProfileSnapshotInput{
			DisplayName: profile.DisplayName,
			Rating:      profile.Rating,
			MaxRating:   profile.MaxRating,
			ProfileURL:  profile.ProfileURL,
			Source:      model.SyncSourceCodeforcesAPI,
			Payload:     profile.Payload,
			FetchedAt:   profile.FetchedAt,
		},
		AcceptedEvents:   toCodeforcesAcceptedEventInputs(acceptedSubmissions),
		ContestHistories: toCodeforcesContestHistoryInputs(contestHistory),
	}); err != nil {
		return CodeforcesSyncResult{}, fmt.Errorf("save codeforces sync: %w", err)
	}

	return CodeforcesSyncResult{
		SyncedAt:            syncedAt,
		AcceptedEventCount:  len(acceptedSubmissions),
		ProblemFactCount:    uniqueProblemFactCount(acceptedSubmissions),
		ContestSummaryCount: uniqueContestSummaryCount(acceptedSubmissions),
		ContestHistoryCount: len(contestHistory),
	}, nil
}

func (s *CodeforcesSyncService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	account, err := s.loadOwnedCodeforcesAccount(ctx, siteUserID, accountID)
	if err != nil {
		return model.PlatformProfileSnapshot{}, err
	}

	snapshot, err := s.syncStore.GetLatestProfileSnapshot(ctx, account.ID)
	if err != nil {
		if errors.Is(err, repository.ErrPlatformSyncDataNotFound) {
			return model.PlatformProfileSnapshot{}, ErrCodeforcesSyncDataNotFound
		}

		return model.PlatformProfileSnapshot{}, fmt.Errorf("load latest codeforces profile: %w", err)
	}

	return snapshot, nil
}

func (s *CodeforcesSyncService) ListContestHistories(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
	input ListCodeforcesSyncInput,
) ([]model.PlatformContestHistory, error) {
	account, err := s.loadOwnedCodeforcesAccount(ctx, siteUserID, accountID)
	if err != nil {
		return nil, err
	}

	filter, err := normalizeListCodeforcesSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListContestHistoriesByAccountID(ctx, account.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("list codeforces contest histories: %w", err)
	}

	return items, nil
}

func (s *CodeforcesSyncService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input ListCodeforcesSyncInput,
) ([]model.ProblemFact, error) {
	filter, err := normalizeListCodeforcesSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListProblemFactsByUserIDAndPlatform(ctx, siteUserID, model.PlatformCodeforces, filter)
	if err != nil {
		return nil, fmt.Errorf("list codeforces problem facts: %w", err)
	}

	return items, nil
}

func (s *CodeforcesSyncService) ListContestSummaries(
	ctx context.Context,
	siteUserID int64,
	input ListCodeforcesSyncInput,
) ([]model.ContestACSummary, error) {
	filter, err := normalizeListCodeforcesSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListContestSummariesByUserIDAndPlatform(ctx, siteUserID, model.PlatformCodeforces, filter)
	if err != nil {
		return nil, fmt.Errorf("list codeforces contest summaries: %w", err)
	}

	return items, nil
}

func (s *CodeforcesSyncService) loadOwnedCodeforcesAccount(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformAccount, error) {
	account, err := s.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountStoreError(err)
	}

	if account.SiteUserID != siteUserID {
		return model.PlatformAccount{}, ErrPlatformAccountForbidden
	}
	if account.Platform != model.PlatformCodeforces {
		return model.PlatformAccount{}, ValidationError{Message: "account must be a codeforces account"}
	}

	return account, nil
}

func normalizeListCodeforcesSyncInput(
	input ListCodeforcesSyncInput,
) (repository.ListCodeforcesSyncFilter, error) {
	switch {
	case input.Limit < 0:
		return repository.ListCodeforcesSyncFilter{}, ValidationError{Message: "limit must be greater than or equal to 0"}
	case input.Offset < 0:
		return repository.ListCodeforcesSyncFilter{}, ValidationError{Message: "offset must be greater than or equal to 0"}
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultCodeforcesListLimit
	}
	if limit > maxCodeforcesListLimit {
		limit = maxCodeforcesListLimit
	}

	return repository.ListCodeforcesSyncFilter{
		Limit:  limit,
		Offset: input.Offset,
	}, nil
}

func toCodeforcesAcceptedEventInputs(
	submissions []integration.CodeforcesAcceptedSubmission,
) []repository.CodeforcesAcceptedEventInput {
	result := make([]repository.CodeforcesAcceptedEventInput, 0, len(submissions))
	for _, submission := range submissions {
		result = append(result, repository.CodeforcesAcceptedEventInput{
			Handle:       submission.Handle,
			ProblemKey:   submission.ProblemKey,
			ContestID:    submission.ContestID,
			ProblemIndex: submission.ProblemIndex,
			ProblemName:  submission.ProblemName,
			ProblemURL:   submission.ProblemURL,
			AcceptedAt:   submission.AcceptedAt,
			SubmissionID: submission.SubmissionID,
			Source:       model.SyncSourceCodeforcesAPI,
			SourceURL:    submission.SourceURL,
			Payload:      submission.Payload,
			FetchedAt:    submission.FetchedAt,
		})
	}

	return result
}

func toCodeforcesContestHistoryInputs(
	history []integration.CodeforcesContestHistoryEntry,
) []repository.CodeforcesContestHistoryInput {
	result := make([]repository.CodeforcesContestHistoryInput, 0, len(history))
	for _, item := range history {
		result = append(result, repository.CodeforcesContestHistoryInput{
			ContestID:      item.ContestID,
			ContestName:    item.ContestName,
			Rank:           item.Rank,
			OldRating:      item.OldRating,
			NewRating:      item.NewRating,
			RatingDelta:    item.RatingDelta,
			ParticipatedAt: item.ParticipatedAt,
			Source:         model.SyncSourceCodeforcesAPI,
			SourceURL:      item.SourceURL,
			Payload:        item.Payload,
			FetchedAt:      item.FetchedAt,
		})
	}

	return result
}

func uniqueProblemFactCount(submissions []integration.CodeforcesAcceptedSubmission) int {
	seen := make(map[string]struct{}, len(submissions))
	for _, submission := range submissions {
		seen[submission.ProblemKey] = struct{}{}
	}

	return len(seen)
}

func uniqueContestSummaryCount(submissions []integration.CodeforcesAcceptedSubmission) int {
	seen := make(map[string]struct{}, len(submissions))
	for _, submission := range submissions {
		if submission.ContestID == "" {
			continue
		}

		seen[submission.ContestID] = struct{}{}
	}

	return len(seen)
}

func mapCodeforcesClientError(err error) error {
	switch {
	case errors.Is(err, integration.ErrCodeforcesUserNotFound):
		return ValidationError{Message: "codeforces handle was not found"}
	case errors.Is(err, integration.ErrCodeforcesAPI):
		return ErrCodeforcesUpstream
	default:
		return fmt.Errorf("codeforces client: %w", err)
	}
}

func syncJobErrorMessage(err error) string {
	return syncJobErrorMessageWithDefault(err, "codeforces sync failed")
}
