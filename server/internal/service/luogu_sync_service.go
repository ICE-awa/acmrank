package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

var (
	ErrLuoguUpstream         = errors.New("luogu upstream unavailable")
	ErrLuoguSyncDataNotFound = errors.New("luogu sync data not found")
)

type LuoguPlatformAccountStore interface {
	GetByID(ctx context.Context, accountID int64) (model.PlatformAccount, error)
}

type LuoguSyncStore interface {
	SaveSync(ctx context.Context, params repository.SaveLuoguSyncParams) error
	GetLatestProfileSnapshot(ctx context.Context, accountID int64) (model.PlatformProfileSnapshot, error)
	ListProblemFactsByUserIDAndPlatform(ctx context.Context, siteUserID int64, platform model.Platform, filter repository.ListPlatformSyncFilter) ([]model.ProblemFact, error)
}

type LuoguSyncJobStore interface {
	Enqueue(ctx context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error)
	ClaimNextQueuedJob(ctx context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error)
	MarkSucceeded(ctx context.Context, jobID int64, finishedAt time.Time) error
	MarkFailed(ctx context.Context, jobID int64, errorMessage string, finishedAt time.Time) error
}

type LuoguSyncClient interface {
	FetchSnapshot(ctx context.Context, handle string) (integration.LuoguSyncSnapshot, error)
}

type LuoguSyncResult struct {
	SyncedAt             time.Time
	AcceptedProblemCount int
	ProblemFactCount     int
}

type LuoguSyncService struct {
	accountStore LuoguPlatformAccountStore
	syncStore    LuoguSyncStore
	jobStore     LuoguSyncJobStore
	client       LuoguSyncClient
	now          func() time.Time
}

func NewLuoguSyncService(
	accountStore LuoguPlatformAccountStore,
	syncStore LuoguSyncStore,
	jobStore LuoguSyncJobStore,
	client LuoguSyncClient,
) *LuoguSyncService {
	return &LuoguSyncService{
		accountStore: accountStore,
		syncStore:    syncStore,
		jobStore:     jobStore,
		client:       client,
		now:          time.Now,
	}
}

func (s *LuoguSyncService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	account, err := s.loadOwnedLuoguAccount(ctx, siteUserID, accountID)
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
		JobType:           model.SyncJobTypeLuogu,
		ScheduledAt:       s.now().UTC(),
	})
	if err != nil {
		return model.SyncJob{}, fmt.Errorf("enqueue luogu sync: %w", err)
	}

	return job, nil
}

func (s *LuoguSyncService) ProcessNextQueuedSync(
	ctx context.Context,
) (bool, error) {
	startedAt := s.now().UTC()
	job, err := s.jobStore.ClaimNextQueuedJob(ctx, model.SyncJobTypeLuogu, startedAt)
	if err != nil {
		if errors.Is(err, repository.ErrNoPendingSyncJob) {
			return false, nil
		}

		return false, fmt.Errorf("claim luogu sync job: %w", err)
	}

	processErr := s.processSyncJob(ctx, job)
	finishedAt := s.now().UTC()
	if processErr != nil {
		if markErr := s.jobStore.MarkFailed(ctx, job.ID, syncJobErrorMessageWithDefault(processErr, "luogu sync failed"), finishedAt); markErr != nil {
			return true, errors.Join(processErr, fmt.Errorf("mark luogu sync job failed: %w", markErr))
		}

		return true, fmt.Errorf("process luogu sync job %d: %w", job.ID, processErr)
	}

	if err := s.jobStore.MarkSucceeded(ctx, job.ID, finishedAt); err != nil {
		return true, fmt.Errorf("mark luogu sync job succeeded: %w", err)
	}

	return true, nil
}

func (s *LuoguSyncService) Sync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (LuoguSyncResult, error) {
	account, err := s.loadOwnedLuoguAccount(ctx, siteUserID, accountID)
	if err != nil {
		return LuoguSyncResult{}, err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return LuoguSyncResult{}, ErrPlatformAccountNotReady
	}

	return s.syncAccount(ctx, account)
}

func (s *LuoguSyncService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	account, err := s.loadOwnedLuoguAccount(ctx, siteUserID, accountID)
	if err != nil {
		return model.PlatformProfileSnapshot{}, err
	}

	snapshot, err := s.syncStore.GetLatestProfileSnapshot(ctx, account.ID)
	if err != nil {
		if errors.Is(err, repository.ErrPlatformSyncDataNotFound) {
			return model.PlatformProfileSnapshot{}, ErrLuoguSyncDataNotFound
		}

		return model.PlatformProfileSnapshot{}, fmt.Errorf("load latest luogu profile: %w", err)
	}

	return snapshot, nil
}

func (s *LuoguSyncService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input ListPlatformSyncInput,
) ([]model.ProblemFact, error) {
	filter, err := normalizeListPlatformSyncInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.syncStore.ListProblemFactsByUserIDAndPlatform(ctx, siteUserID, model.PlatformLuogu, filter)
	if err != nil {
		return nil, fmt.Errorf("list luogu problem facts: %w", err)
	}

	return items, nil
}

func (s *LuoguSyncService) processSyncJob(
	ctx context.Context,
	job model.SyncJob,
) error {
	if job.PlatformAccountID == nil {
		return errors.New("sync job is missing platform account id")
	}
	if job.SiteUserID == nil {
		return errors.New("sync job is missing site user id")
	}

	account, err := s.loadOwnedLuoguAccount(ctx, *job.SiteUserID, *job.PlatformAccountID)
	if err != nil {
		return err
	}
	if account.Status != model.PlatformAccountStatusVerified {
		return ErrPlatformAccountNotReady
	}

	_, err = s.syncAccount(ctx, account)
	return err
}

func (s *LuoguSyncService) syncAccount(
	ctx context.Context,
	account model.PlatformAccount,
) (LuoguSyncResult, error) {
	snapshot, err := s.client.FetchSnapshot(ctx, account.Handle)
	if err != nil {
		return LuoguSyncResult{}, mapLuoguClientError(err)
	}

	syncedAt := s.now().UTC()
	acceptedProblems := toLuoguAcceptedProblemInputs(snapshot.AcceptedProblems, syncedAt)
	if err := s.syncStore.SaveSync(ctx, repository.SaveLuoguSyncParams{
		Account:  account,
		SyncedAt: syncedAt,
		Profile: repository.LuoguProfileSnapshotInput{
			DisplayName: snapshot.Profile.DisplayName,
			Rating:      snapshot.Profile.Rating,
			ProfileURL:  snapshot.Profile.ProfileURL,
			Source:      model.SyncSourceLuoguUserInfo,
			Payload:     snapshot.Profile.Payload,
			FetchedAt:   snapshot.Profile.FetchedAt,
		},
		AcceptedProblems: acceptedProblems,
	}); err != nil {
		return LuoguSyncResult{}, fmt.Errorf("save luogu sync: %w", err)
	}

	return LuoguSyncResult{
		SyncedAt:             syncedAt,
		AcceptedProblemCount: len(snapshot.AcceptedProblems),
		ProblemFactCount:     len(snapshot.AcceptedProblems),
	}, nil
}

func (s *LuoguSyncService) loadOwnedLuoguAccount(
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
	if account.Platform != model.PlatformLuogu {
		return model.PlatformAccount{}, ValidationError{Message: "account must be a luogu account"}
	}

	return account, nil
}

func toLuoguAcceptedProblemInputs(
	problems []integration.LuoguAcceptedProblem,
	observedAt time.Time,
) []repository.LuoguAcceptedProblemInput {
	result := make([]repository.LuoguAcceptedProblemInput, 0, len(problems))
	for _, problem := range problems {
		result = append(result, repository.LuoguAcceptedProblemInput{
			Handle:      problem.Handle,
			ProblemKey:  problem.ProblemKey,
			ProblemID:   problem.ProblemID,
			ProblemName: problem.ProblemName,
			ProblemURL:  problem.ProblemURL,
			AcceptedAt:  observedAt,
			Source:      model.SyncSourceLuoguPractice,
			SourceURL:   problem.SourceURL,
			Payload:     problem.Payload,
			FetchedAt:   problem.FetchedAt,
		})
	}

	return result
}

func mapLuoguClientError(err error) error {
	switch {
	case errors.Is(err, integration.ErrLuoguUserNotFound):
		return ValidationError{Message: "luogu handle was not found"}
	case errors.Is(err, integration.ErrLuoguAPI):
		return ErrLuoguUpstream
	default:
		return fmt.Errorf("luogu client: %w", err)
	}
}

func syncJobErrorMessageWithDefault(err error, fallback string) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fallback
	}

	if len(message) <= 512 {
		return message
	}

	return message[:512]
}
