package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
	"golang.org/x/sync/errgroup"
)

var (
	ErrAtCoderUpstream         = errors.New("atcoder upstream unavailable")
	ErrAtCoderSyncDataNotFound = errors.New("atcoder sync data not found")
)

type AtCoderPlatformAccountStore interface {
	GetByID(ctx context.Context, accountID int64) (model.PlatformAccount, error)
}

type AtCoderSyncStore interface {
	SaveSync(ctx context.Context, params repository.SaveAtCoderSyncParams) error
	GetLatestProfileSnapshot(ctx context.Context, accountID int64) (model.PlatformProfileSnapshot, error)
	ListContestHistoriesByAccountID(ctx context.Context, accountID int64, filter repository.ListPlatformSyncFilter) ([]model.PlatformContestHistory, error)
	ListProblemFactsByUserIDAndPlatform(ctx context.Context, siteUserID int64, platform model.Platform, filter repository.ListPlatformSyncFilter) ([]model.ProblemFact, error)
	ListContestSummariesByUserIDAndPlatform(ctx context.Context, siteUserID int64, platform model.Platform, filter repository.ListPlatformSyncFilter) ([]model.ContestACSummary, error)
}

type AtCoderSyncJobStore interface {
	Enqueue(ctx context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error)
	ClaimNextQueuedJob(ctx context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error)
	MarkSucceeded(ctx context.Context, jobID int64, finishedAt time.Time) error
	MarkFailed(ctx context.Context, jobID int64, errorMessage string, finishedAt time.Time) error
}

type AtCoderAlertStore interface {
	RecordFailure(ctx context.Context, params repository.RecordIntegrationAlertParams) error
	Resolve(ctx context.Context, integration model.IntegrationName, resolvedAt time.Time) error
}

type AtCoderSyncClient interface {
	CheckReady(ctx context.Context) error
	FetchProfile(ctx context.Context, handle string) (integration.AtCoderProfile, error)
	FetchContestHistory(ctx context.Context, handle string) ([]integration.AtCoderContestHistoryEntry, error)
	FetchAcceptedSubmissions(ctx context.Context, handle string, contestIDs []string) ([]integration.AtCoderAcceptedSubmission, error)
}

type AtCoderSyncResult struct {
	SyncedAt            time.Time
	AcceptedEventCount  int
	ProblemFactCount    int
	ContestSummaryCount int
	ContestHistoryCount int
}

type AtCoderSyncService struct {
	accountStore AtCoderPlatformAccountStore
	syncStore    AtCoderSyncStore
	jobStore     AtCoderSyncJobStore
	alertStore   AtCoderAlertStore
	client       AtCoderSyncClient
	now          func() time.Time
}

func NewAtCoderSyncService(
	accountStore AtCoderPlatformAccountStore,
	syncStore AtCoderSyncStore,
	jobStore AtCoderSyncJobStore,
	alertStore AtCoderAlertStore,
	client AtCoderSyncClient,
) *AtCoderSyncService {
	return &AtCoderSyncService{
		accountStore: accountStore,
		syncStore:    syncStore,
		jobStore:     jobStore,
		alertStore:   alertStore,
		client:       client,
		now:          time.Now,
	}
}

func (s *AtCoderSyncService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	account, err := s.loadOwnedAtCoderAccount(ctx, siteUserID, accountID)
	if err != nil {
		return model.SyncJob{}, err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return model.SyncJob{}, ErrPlatformAccountNotReady
	}

	if err := s.client.CheckReady(ctx); err != nil {
		return model.SyncJob{}, s.handleAtCoderFailure(ctx, err)
	}

	jobAccountID := account.ID
	job, err := s.jobStore.Enqueue(ctx, repository.EnqueueSyncJobParams{
		SiteUserID:        siteUserID,
		PlatformAccountID: &jobAccountID,
		Platform:          string(account.Platform),
		JobType:           model.SyncJobTypeAtCoder,
		ScheduledAt:       s.now().UTC(),
	})
	if err != nil {
		return model.SyncJob{}, fmt.Errorf("enqueue atcoder sync: %w", err)
	}

	return job, nil
}

func (s *AtCoderSyncService) ProcessNextQueuedSync(
	ctx context.Context,
) (bool, error) {
	startedAt := s.now().UTC()
	job, err := s.jobStore.ClaimNextQueuedJob(ctx, model.SyncJobTypeAtCoder, startedAt)
	if err != nil {
		if errors.Is(err, repository.ErrNoPendingSyncJob) {
			return false, nil
		}

		return false, fmt.Errorf("claim atcoder sync job: %w", err)
	}

	processErr := s.processSyncJob(ctx, job)
	finishedAt := s.now().UTC()
	if processErr != nil {
		if markErr := s.jobStore.MarkFailed(ctx, job.ID, syncJobErrorMessageWithDefault(processErr, "atcoder sync failed"), finishedAt); markErr != nil {
			return true, errors.Join(processErr, fmt.Errorf("mark atcoder sync job failed: %w", markErr))
		}

		return true, fmt.Errorf("process atcoder sync job %d: %w", job.ID, processErr)
	}

	if err := s.jobStore.MarkSucceeded(ctx, job.ID, finishedAt); err != nil {
		return true, fmt.Errorf("mark atcoder sync job succeeded: %w", err)
	}

	return true, nil
}

func (s *AtCoderSyncService) Sync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (AtCoderSyncResult, error) {
	account, err := s.loadOwnedAtCoderAccount(ctx, siteUserID, accountID)
	if err != nil {
		return AtCoderSyncResult{}, err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return AtCoderSyncResult{}, ErrPlatformAccountNotReady
	}

	return s.syncAccount(ctx, account)
}

func (s *AtCoderSyncService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	account, err := s.loadOwnedAtCoderAccount(ctx, siteUserID, accountID)
	if err != nil {
		return model.PlatformProfileSnapshot{}, err
	}

	snapshot, err := s.syncStore.GetLatestProfileSnapshot(ctx, account.ID)
	if err != nil {
		if errors.Is(err, repository.ErrPlatformSyncDataNotFound) {
			return model.PlatformProfileSnapshot{}, ErrAtCoderSyncDataNotFound
		}

		return model.PlatformProfileSnapshot{}, fmt.Errorf("load latest atcoder profile: %w", err)
	}

	return snapshot, nil
}

func (s *AtCoderSyncService) ListContestHistories(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
	input ListPlatformSyncInput,
) ([]model.PlatformContestHistory, error) {
	account, err := s.loadOwnedAtCoderAccount(ctx, siteUserID, accountID)
	if err != nil {
		return nil, err
	}

	filter, err := normalizeListPlatformSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListContestHistoriesByAccountID(ctx, account.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("list atcoder contest histories: %w", err)
	}

	return items, nil
}

func (s *AtCoderSyncService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input ListPlatformSyncInput,
) ([]model.ProblemFact, error) {
	filter, err := normalizeListPlatformSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListProblemFactsByUserIDAndPlatform(ctx, siteUserID, model.PlatformAtCoder, filter)
	if err != nil {
		return nil, fmt.Errorf("list atcoder problem facts: %w", err)
	}

	return items, nil
}

func (s *AtCoderSyncService) ListContestSummaries(
	ctx context.Context,
	siteUserID int64,
	input ListPlatformSyncInput,
) ([]model.ContestACSummary, error) {
	filter, err := normalizeListPlatformSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListContestSummariesByUserIDAndPlatform(ctx, siteUserID, model.PlatformAtCoder, filter)
	if err != nil {
		return nil, fmt.Errorf("list atcoder contest summaries: %w", err)
	}

	return items, nil
}

func (s *AtCoderSyncService) processSyncJob(
	ctx context.Context,
	job model.SyncJob,
) error {
	if job.PlatformAccountID == nil {
		return errors.New("sync job is missing platform account id")
	}
	if job.SiteUserID == nil {
		return errors.New("sync job is missing site user id")
	}

	account, err := s.loadOwnedAtCoderAccount(ctx, *job.SiteUserID, *job.PlatformAccountID)
	if err != nil {
		return err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return ErrPlatformAccountNotReady
	}

	_, err = s.syncAccount(ctx, account)
	return err
}

func (s *AtCoderSyncService) syncAccount(
	ctx context.Context,
	account model.PlatformAccount,
) (AtCoderSyncResult, error) {
	if err := s.client.CheckReady(ctx); err != nil {
		return AtCoderSyncResult{}, s.handleAtCoderFailure(ctx, err)
	}

	var profile integration.AtCoderProfile
	var contestHistory []integration.AtCoderContestHistoryEntry

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var fetchErr error
		profile, fetchErr = s.client.FetchProfile(groupCtx, account.Handle)
		return fetchErr
	})
	group.Go(func() error {
		var fetchErr error
		contestHistory, fetchErr = s.client.FetchContestHistory(groupCtx, account.Handle)
		return fetchErr
	})

	if err := group.Wait(); err != nil {
		return AtCoderSyncResult{}, s.handleAtCoderFailure(ctx, err)
	}

	acceptedSubmissions, err := s.client.FetchAcceptedSubmissions(
		ctx,
		account.Handle,
		atCoderContestIDs(contestHistory),
	)
	if err != nil {
		return AtCoderSyncResult{}, s.handleAtCoderFailure(ctx, err)
	}

	syncedAt := s.now().UTC()
	if err := s.syncStore.SaveSync(ctx, repository.SaveAtCoderSyncParams{
		Account:  account,
		SyncedAt: syncedAt,
		Profile: repository.PlatformProfileSnapshotInput{
			DisplayName: profile.DisplayName,
			Rating:      profile.Rating,
			MaxRating:   profile.MaxRating,
			ProfileURL:  profile.ProfileURL,
			Source:      model.SyncSourceAtCoderProfilePage,
			Payload:     profile.Payload,
			FetchedAt:   profile.FetchedAt,
		},
		AcceptedEvents:   toAtCoderAcceptedEventInputs(acceptedSubmissions),
		ContestHistories: toAtCoderContestHistoryInputs(contestHistory),
	}); err != nil {
		if alertErr := s.recordMainFailure(ctx, err); alertErr != nil {
			return AtCoderSyncResult{}, errors.Join(fmt.Errorf("save atcoder sync: %w", err), fmt.Errorf("record atcoder alert: %w", alertErr))
		}

		return AtCoderSyncResult{}, fmt.Errorf("save atcoder sync: %w", err)
	}

	if err := s.alertStore.Resolve(ctx, model.IntegrationAtCoderMain, syncedAt); err != nil {
		log.Printf("resolve atcoder main alert error: %v", err)
	}

	return AtCoderSyncResult{
		SyncedAt:            syncedAt,
		AcceptedEventCount:  len(acceptedSubmissions),
		ProblemFactCount:    uniqueAtCoderProblemFactCount(acceptedSubmissions),
		ContestSummaryCount: uniqueAtCoderContestSummaryCount(acceptedSubmissions),
		ContestHistoryCount: len(contestHistory),
	}, nil
}

func (s *AtCoderSyncService) handleAtCoderFailure(
	ctx context.Context,
	err error,
) error {
	mappedErr := mapAtCoderClientError(err)
	if !shouldRecordAtCoderAlert(err) {
		return mappedErr
	}

	if alertErr := s.recordMainFailure(ctx, err); alertErr != nil {
		return errors.Join(mappedErr, fmt.Errorf("record atcoder alert: %w", alertErr))
	}

	return mappedErr
}

func (s *AtCoderSyncService) recordMainFailure(
	ctx context.Context,
	err error,
) error {
	return s.alertStore.RecordFailure(ctx, repository.RecordIntegrationAlertParams{
		Integration: model.IntegrationAtCoderMain,
		Summary:     "AtCoder main chain sync failed",
		Detail:      syncJobErrorMessageWithDefault(err, "atcoder main chain sync failed"),
		TriggeredAt: s.now().UTC(),
	})
}

func (s *AtCoderSyncService) loadOwnedAtCoderAccount(
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
	if account.Platform != model.PlatformAtCoder {
		return model.PlatformAccount{}, ValidationError{Message: "account must be an atcoder account"}
	}

	return account, nil
}

func atCoderContestIDs(history []integration.AtCoderContestHistoryEntry) []string {
	result := make([]string, 0, len(history))
	for _, item := range history {
		if item.ContestID == "" {
			continue
		}

		result = append(result, item.ContestID)
	}

	return result
}

func toAtCoderAcceptedEventInputs(
	submissions []integration.AtCoderAcceptedSubmission,
) []repository.PlatformAcceptedEventInput {
	result := make([]repository.PlatformAcceptedEventInput, 0, len(submissions))
	for _, submission := range submissions {
		result = append(result, repository.PlatformAcceptedEventInput{
			Handle:       submission.Handle,
			ProblemKey:   submission.ProblemKey,
			ContestID:    submission.ContestID,
			ProblemIndex: submission.TaskID,
			ProblemName:  submission.ProblemName,
			ProblemURL:   submission.ProblemURL,
			AcceptedAt:   submission.AcceptedAt,
			SubmissionID: submission.SubmissionID,
			Source:       model.SyncSourceAtCoderSubmission,
			SourceURL:    submission.SourceURL,
			Payload:      submission.Payload,
			FetchedAt:    submission.FetchedAt,
		})
	}

	return result
}

func toAtCoderContestHistoryInputs(
	history []integration.AtCoderContestHistoryEntry,
) []repository.AtCoderContestHistoryInput {
	result := make([]repository.AtCoderContestHistoryInput, 0, len(history))
	for _, item := range history {
		result = append(result, repository.AtCoderContestHistoryInput{
			ContestID:      item.ContestID,
			ContestName:    item.ContestName,
			Rank:           item.Rank,
			OldRating:      item.OldRating,
			NewRating:      item.NewRating,
			RatingDelta:    item.RatingDelta,
			ParticipatedAt: item.ParticipatedAt,
			Source:         model.SyncSourceAtCoderHistoryJSON,
			SourceURL:      item.SourceURL,
			Payload:        item.Payload,
			FetchedAt:      item.FetchedAt,
		})
	}

	return result
}

func uniqueAtCoderProblemFactCount(submissions []integration.AtCoderAcceptedSubmission) int {
	seen := make(map[string]struct{}, len(submissions))
	for _, submission := range submissions {
		seen[submission.ProblemKey] = struct{}{}
	}

	return len(seen)
}

func uniqueAtCoderContestSummaryCount(submissions []integration.AtCoderAcceptedSubmission) int {
	seen := make(map[string]struct{}, len(submissions))
	for _, submission := range submissions {
		if submission.ContestID == "" {
			continue
		}

		seen[submission.ContestID] = struct{}{}
	}

	return len(seen)
}

func shouldRecordAtCoderAlert(err error) bool {
	return !errors.Is(err, integration.ErrAtCoderUserNotFound) &&
		!errors.Is(err, ErrValidation)
}

func mapAtCoderClientError(err error) error {
	switch {
	case errors.Is(err, integration.ErrAtCoderUserNotFound):
		return ValidationError{Message: "atcoder handle was not found"}
	case errors.Is(err, integration.ErrAtCoderAPI),
		errors.Is(err, integration.ErrAtCoderCredentialMissing):
		return ErrAtCoderUpstream
	default:
		return fmt.Errorf("atcoder client: %w", err)
	}
}
