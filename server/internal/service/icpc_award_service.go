package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

var ErrICPCAwardUpstream = errors.New("icpc awards upstream unavailable")

const (
	defaultAwardRecordListLimit = 50
	maxAwardRecordListLimit     = 100
)

type ICPCAwardUserStore interface {
	GetByID(ctx context.Context, id int64) (model.User, error)
}

type ICPCAwardStore interface {
	ReplaceAutoSyncByUserID(ctx context.Context, params repository.ReplaceAwardRecordsParams) error
	ListByUserID(ctx context.Context, siteUserID int64, filter repository.ListAwardRecordsFilter) ([]model.AwardRecord, error)
}

type ICPCAwardJobStore interface {
	Enqueue(ctx context.Context, params repository.EnqueueSyncJobParams) (model.SyncJob, error)
	ClaimNextQueuedJob(ctx context.Context, jobType model.SyncJobType, startedAt time.Time) (model.SyncJob, error)
	MarkSucceeded(ctx context.Context, jobID int64, finishedAt time.Time) error
	MarkFailed(ctx context.Context, jobID int64, errorMessage string, finishedAt time.Time) error
}

type ICPCAwardClient interface {
	FetchAwards(ctx context.Context) ([]integration.ICPCAwardFeedRecord, error)
}

type ListAwardRecordsInput struct {
	Limit  int
	Offset int
}

type ICPCAwardSyncResult struct {
	SyncedAt          time.Time
	MatchedAwardCount int
}

type ICPCAwardService struct {
	userStore  ICPCAwardUserStore
	awardStore ICPCAwardStore
	jobStore   ICPCAwardJobStore
	client     ICPCAwardClient
	now        func() time.Time
}

func NewICPCAwardService(
	userStore ICPCAwardUserStore,
	awardStore ICPCAwardStore,
	jobStore ICPCAwardJobStore,
	client ICPCAwardClient,
) *ICPCAwardService {
	return &ICPCAwardService{
		userStore:  userStore,
		awardStore: awardStore,
		jobStore:   jobStore,
		client:     client,
		now:        time.Now,
	}
}

func (s *ICPCAwardService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
) (model.SyncJob, error) {
	if _, err := s.loadUser(ctx, siteUserID); err != nil {
		return model.SyncJob{}, err
	}

	job, err := s.jobStore.Enqueue(ctx, repository.EnqueueSyncJobParams{
		SiteUserID:        siteUserID,
		PlatformAccountID: nil,
		Platform:          model.AwardPlatformICPC,
		JobType:           model.SyncJobTypeICPCAward,
		ScheduledAt:       s.now().UTC(),
	})
	if err != nil {
		return model.SyncJob{}, fmt.Errorf("enqueue icpc award sync: %w", err)
	}

	return job, nil
}

func (s *ICPCAwardService) ProcessNextQueuedSync(
	ctx context.Context,
) (bool, error) {
	startedAt := s.now().UTC()
	job, err := s.jobStore.ClaimNextQueuedJob(ctx, model.SyncJobTypeICPCAward, startedAt)
	if err != nil {
		if errors.Is(err, repository.ErrNoPendingSyncJob) {
			return false, nil
		}

		return false, fmt.Errorf("claim icpc award sync job: %w", err)
	}

	processErr := s.processSyncJob(ctx, job)
	finishedAt := s.now().UTC()
	if processErr != nil {
		if markErr := s.jobStore.MarkFailed(ctx, job.ID, syncJobErrorMessageWithDefault(processErr, "icpc award sync failed"), finishedAt); markErr != nil {
			return true, errors.Join(processErr, fmt.Errorf("mark icpc award sync job failed: %w", markErr))
		}

		return true, fmt.Errorf("process icpc award sync job %d: %w", job.ID, processErr)
	}

	if err := s.jobStore.MarkSucceeded(ctx, job.ID, finishedAt); err != nil {
		return true, fmt.Errorf("mark icpc award sync job succeeded: %w", err)
	}

	return true, nil
}

func (s *ICPCAwardService) Sync(
	ctx context.Context,
	siteUserID int64,
) (ICPCAwardSyncResult, error) {
	user, err := s.loadUser(ctx, siteUserID)
	if err != nil {
		return ICPCAwardSyncResult{}, err
	}

	if strings.TrimSpace(user.RealName) == "" {
		return ICPCAwardSyncResult{}, ValidationError{Message: "real_name is required before syncing awards"}
	}

	records, err := s.client.FetchAwards(ctx)
	if err != nil {
		return ICPCAwardSyncResult{}, mapICPCAwardClientError(err)
	}

	matched := toAwardRecordInputs(user.RealName, records)
	syncedAt := s.now().UTC()
	if err := s.awardStore.ReplaceAutoSyncByUserID(ctx, repository.ReplaceAwardRecordsParams{
		SiteUserID: user.ID,
		Platform:   model.AwardPlatformICPC,
		Records:    matched,
	}); err != nil {
		return ICPCAwardSyncResult{}, fmt.Errorf("replace icpc awards: %w", err)
	}

	return ICPCAwardSyncResult{
		SyncedAt:          syncedAt,
		MatchedAwardCount: len(matched),
	}, nil
}

func (s *ICPCAwardService) ListAwards(
	ctx context.Context,
	siteUserID int64,
	input ListAwardRecordsInput,
) ([]model.AwardRecord, error) {
	filter, err := normalizeListAwardRecordsInput(input)
	if err != nil {
		return nil, err
	}

	items, err := s.awardStore.ListByUserID(ctx, siteUserID, filter)
	if err != nil {
		return nil, fmt.Errorf("list award records: %w", err)
	}

	return items, nil
}

func (s *ICPCAwardService) processSyncJob(
	ctx context.Context,
	job model.SyncJob,
) error {
	if job.SiteUserID == nil {
		return errors.New("sync job is missing site user id")
	}

	_, err := s.Sync(ctx, *job.SiteUserID)
	return err
}

func (s *ICPCAwardService) loadUser(
	ctx context.Context,
	siteUserID int64,
) (model.User, error) {
	user, err := s.userStore.GetByID(ctx, siteUserID)
	if err != nil {
		return model.User{}, fmt.Errorf("load award user: %w", err)
	}

	return user, nil
}

func normalizeListAwardRecordsInput(
	input ListAwardRecordsInput,
) (repository.ListAwardRecordsFilter, error) {
	switch {
	case input.Limit < 0:
		return repository.ListAwardRecordsFilter{}, ValidationError{
			Message: "limit must be greater than or equal to 0",
		}
	case input.Offset < 0:
		return repository.ListAwardRecordsFilter{}, ValidationError{
			Message: "offset must be greater than or equal to 0",
		}
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultAwardRecordListLimit
	}
	if limit > maxAwardRecordListLimit {
		limit = maxAwardRecordListLimit
	}

	return repository.ListAwardRecordsFilter{
		Limit:  limit,
		Offset: input.Offset,
	}, nil
}

func toAwardRecordInputs(
	realName string,
	records []integration.ICPCAwardFeedRecord,
) []repository.AwardRecordInput {
	normalizedRealName := normalizePersonName(realName)
	if normalizedRealName == "" {
		return nil
	}

	seen := make(map[string]struct{}, len(records))
	inputs := make([]repository.AwardRecordInput, 0, len(records))
	for _, record := range records {
		if !awardRecordMatchesRealName(normalizedRealName, record) {
			continue
		}

		key := record.ContestName + "\x00" + record.AwardName + "\x00" + record.AwardDate.Format(time.DateOnly)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		inputs = append(inputs, repository.AwardRecordInput{
			ContestName: record.ContestName,
			AwardName:   record.AwardName,
			RankText:    record.RankText,
			AwardDate:   record.AwardDate,
			Source:      model.SyncSourceICPCAwardsFeed,
			SourceURL:   record.SourceURL,
			Notes:       record.Notes,
		})
	}

	return inputs
}

func awardRecordMatchesRealName(
	normalizedRealName string,
	record integration.ICPCAwardFeedRecord,
) bool {
	for _, member := range record.Members {
		if normalizePersonName(member) == normalizedRealName {
			return true
		}
	}

	return false
}

func normalizePersonName(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case unicode.IsSpace(r), unicode.IsPunct(r), unicode.IsSymbol(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func mapICPCAwardClientError(err error) error {
	switch {
	case errors.Is(err, integration.ErrICPCAwardsAPI):
		return ErrICPCAwardUpstream
	default:
		return fmt.Errorf("icpc awards client: %w", err)
	}
}
