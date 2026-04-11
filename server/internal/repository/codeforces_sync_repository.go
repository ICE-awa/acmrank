package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
)

type CodeforcesContestHistoryInput struct {
	ContestID      string
	ContestName    string
	Rank           int
	OldRating      int
	NewRating      int
	RatingDelta    int
	ParticipatedAt time.Time
	Source         string
	SourceURL      string
	Payload        []byte
	FetchedAt      time.Time
}

type SaveCodeforcesSyncParams struct {
	Account          model.PlatformAccount
	SyncedAt         time.Time
	Profile          PlatformProfileSnapshotInput
	AcceptedEvents   []PlatformAcceptedEventInput
	ContestHistories []CodeforcesContestHistoryInput
}

type CodeforcesSyncRepository struct {
	*PlatformSyncRepository
}

const codeforcesSyncBatchSize = 500

func NewCodeforcesSyncRepository(db platformSyncRepositoryDB) *CodeforcesSyncRepository {
	return &CodeforcesSyncRepository{
		PlatformSyncRepository: NewPlatformSyncRepository(db),
	}
}

func (r *CodeforcesSyncRepository) SaveSync(
	ctx context.Context,
	params SaveCodeforcesSyncParams,
) error {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := r.insertProfileSnapshot(ctx, tx, params.Account.ID, params.Profile); err != nil {
		return err
	}

	contestNameByID := make(map[string]string, len(params.ContestHistories))
	if err := r.upsertContestHistoriesBatch(ctx, tx, params.Account, params.ContestHistories); err != nil {
		return err
	}
	for _, contest := range params.ContestHistories {
		contestNameByID[contest.ContestID] = contest.ContestName
	}

	sortedAccepted := append([]PlatformAcceptedEventInput(nil), params.AcceptedEvents...)
	sort.Slice(sortedAccepted, func(i, j int) bool {
		if sortedAccepted[i].AcceptedAt.Equal(sortedAccepted[j].AcceptedAt) {
			return sortedAccepted[i].SubmissionID < sortedAccepted[j].SubmissionID
		}

		return sortedAccepted[i].AcceptedAt.Before(sortedAccepted[j].AcceptedAt)
	})

	problemFacts := make(map[string]problemFactAggregate)
	contestSummaries := make(map[string]contestSummaryAggregate)
	insertedAcceptedEvents, err := r.insertAcceptedEventRawBatch(ctx, tx, params.Account, sortedAccepted)
	if err != nil {
		return err
	}
	rawIDByAcceptedEvent := make(map[string]int64, len(insertedAcceptedEvents))
	for _, insertedAcceptedEvent := range insertedAcceptedEvents {
		rawIDByAcceptedEvent[acceptedEventIdentity(
			insertedAcceptedEvent.SubmissionID,
			insertedAcceptedEvent.ProblemKey,
			insertedAcceptedEvent.AcceptedAt,
		)] = insertedAcceptedEvent.ID
	}

	for _, acceptedEvent := range sortedAccepted {
		rawID, ok := rawIDByAcceptedEvent[acceptedEventIdentity(
			acceptedEvent.SubmissionID,
			acceptedEvent.ProblemKey,
			acceptedEvent.AcceptedAt,
		)]
		if !ok {
			return fmt.Errorf("accepted event raw id missing for submission %q", acceptedEvent.SubmissionID)
		}

		aggregateProblemFact(problemFacts, acceptedEvent, rawID)
		aggregateContestSummary(contestSummaries, acceptedEvent, contestNameByID[acceptedEvent.ContestID])
	}

	problemKeys := make([]string, 0, len(problemFacts))
	for problemKey := range problemFacts {
		problemKeys = append(problemKeys, problemKey)
	}
	sort.Strings(problemKeys)

	existingProblemFacts, err := r.loadExistingProblemFactsByKeys(ctx, tx, params.Account, problemKeys)
	if err != nil {
		return err
	}

	mergedProblemFacts := make([]problemFactAggregate, 0, len(problemKeys))
	for _, problemKey := range problemKeys {
		mergedProblemFacts = append(mergedProblemFacts, mergeProblemFact(
			existingProblemFacts[problemKey],
			problemFacts[problemKey],
		))
	}
	if err := r.upsertProblemFactsBatch(ctx, tx, params.Account, mergedProblemFacts); err != nil {
		return err
	}

	contestIDs := make([]string, 0, len(contestSummaries))
	for contestID := range contestSummaries {
		contestIDs = append(contestIDs, contestID)
	}
	sort.Strings(contestIDs)

	existingContestSummaries, err := r.loadExistingContestSummariesByIDs(ctx, tx, params.Account, contestIDs)
	if err != nil {
		return err
	}

	mergedContestSummaries := make([]contestSummaryAggregate, 0, len(contestIDs))
	for _, contestID := range contestIDs {
		mergedContestSummaries = append(mergedContestSummaries, mergeContestSummary(
			existingContestSummaries[contestID],
			contestSummaries[contestID],
		))
	}
	if err := r.upsertContestSummariesBatch(ctx, tx, params.Account, mergedContestSummaries); err != nil {
		return err
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE platform_accounts
SET last_synced_at = $2,
    updated_at = $2
WHERE id = $1`,
		params.Account.ID,
		params.SyncedAt.UTC(),
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	committed = true
	return nil
}

func (r *PlatformSyncRepository) GetLatestProfileSnapshot(
	ctx context.Context,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	snapshot, err := scanPlatformProfileSnapshot(
		r.db.QueryRow(
			ctx,
			`SELECT id, platform_account_id, source, COALESCE(display_name, ''), rating, max_rating,
       COALESCE(profile_url, ''), COALESCE(raw_payload_ref, ''), fetched_at, created_at
FROM platform_profile_snapshots
WHERE platform_account_id = $1
ORDER BY fetched_at DESC, id DESC
LIMIT 1`,
			accountID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.PlatformProfileSnapshot{}, ErrPlatformSyncDataNotFound
		}

		return model.PlatformProfileSnapshot{}, err
	}

	return snapshot, nil
}

func (r *CodeforcesSyncRepository) ListContestHistoriesByAccountID(
	ctx context.Context,
	accountID int64,
	filter ListPlatformSyncFilter,
) ([]model.PlatformContestHistory, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, site_user_id, platform_account_id, platform, contest_id, contest_name, rank,
       old_rating, new_rating, rating_delta, participated_at, source, COALESCE(source_url, ''),
       COALESCE(raw_payload_ref, ''), fetched_at, created_at, updated_at
FROM platform_contest_histories
WHERE platform_account_id = $1
ORDER BY participated_at DESC, id DESC
LIMIT $2 OFFSET $3`,
		accountID,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectPlatformContestHistories(rows)
}

func (r *PlatformSyncRepository) ListProblemFactsByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter ListPlatformSyncFilter,
) ([]model.ProblemFact, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, site_user_id, platform, problem_key, COALESCE(contest_id, ''),
       COALESCE(problem_index_or_task_id, ''), COALESCE(problem_name, ''), COALESCE(problem_url, ''),
       first_ac_at, first_ac_source, COALESCE(first_ac_submission_ref, ''), first_ac_event_raw_id,
       latest_ac_at, clist_problem_id, clist_contest_id, clist_rating, created_at, updated_at
FROM problem_facts
WHERE site_user_id = $1
  AND platform = $2
ORDER BY first_ac_at DESC, id DESC
LIMIT $3 OFFSET $4`,
		siteUserID,
		platform,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectProblemFacts(rows)
}

func (r *PlatformSyncRepository) ListContestSummariesByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter ListPlatformSyncFilter,
) ([]model.ContestACSummary, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, site_user_id, platform, contest_id, COALESCE(contest_name, ''), ac_problem_keys,
       ac_count, first_ac_at, last_ac_at, created_at, updated_at
FROM contest_ac_summaries
WHERE site_user_id = $1
  AND platform = $2
ORDER BY COALESCE(last_ac_at, first_ac_at) DESC NULLS LAST, id DESC
LIMIT $3 OFFSET $4`,
		siteUserID,
		platform,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectContestACSummaries(rows)
}

type problemFactAggregate struct {
	ProblemKey          string
	ContestID           string
	ProblemIndex        string
	ProblemName         string
	ProblemURL          string
	FirstACAt           time.Time
	FirstACSource       string
	FirstACSubmissionID string
	FirstACEventRawID   *int64
	LatestACAt          time.Time
}

type contestSummaryAggregate struct {
	ContestID     string
	ContestName   string
	ACProblemKeys map[string]struct{}
	FirstACAt     time.Time
	LastACAt      time.Time
}

type insertedAcceptedEventRaw struct {
	ID           int64
	ProblemKey   string
	AcceptedAt   time.Time
	SubmissionID string
}

func aggregateProblemFact(
	aggregates map[string]problemFactAggregate,
	acceptedEvent PlatformAcceptedEventInput,
	rawID int64,
) {
	current, exists := aggregates[acceptedEvent.ProblemKey]
	if !exists {
		firstACEventRawID := rawID
		aggregates[acceptedEvent.ProblemKey] = problemFactAggregate{
			ProblemKey:          acceptedEvent.ProblemKey,
			ContestID:           acceptedEvent.ContestID,
			ProblemIndex:        acceptedEvent.ProblemIndex,
			ProblemName:         acceptedEvent.ProblemName,
			ProblemURL:          acceptedEvent.ProblemURL,
			FirstACAt:           acceptedEvent.AcceptedAt,
			FirstACSource:       acceptedEvent.Source,
			FirstACSubmissionID: acceptedEvent.SubmissionID,
			FirstACEventRawID:   &firstACEventRawID,
			LatestACAt:          acceptedEvent.AcceptedAt,
		}
		return
	}

	if acceptedEvent.AcceptedAt.Before(current.FirstACAt) {
		firstACEventRawID := rawID
		current.FirstACAt = acceptedEvent.AcceptedAt
		current.FirstACSource = acceptedEvent.Source
		current.FirstACSubmissionID = acceptedEvent.SubmissionID
		current.FirstACEventRawID = &firstACEventRawID
	}
	if acceptedEvent.AcceptedAt.After(current.LatestACAt) {
		current.LatestACAt = acceptedEvent.AcceptedAt
	}

	aggregates[acceptedEvent.ProblemKey] = current
}

func aggregateContestSummary(
	aggregates map[string]contestSummaryAggregate,
	acceptedEvent PlatformAcceptedEventInput,
	contestName string,
) {
	current, exists := aggregates[acceptedEvent.ContestID]
	if !exists {
		current = contestSummaryAggregate{
			ContestID:     acceptedEvent.ContestID,
			ContestName:   contestName,
			ACProblemKeys: map[string]struct{}{acceptedEvent.ProblemKey: {}},
			FirstACAt:     acceptedEvent.AcceptedAt,
			LastACAt:      acceptedEvent.AcceptedAt,
		}
		aggregates[acceptedEvent.ContestID] = current
		return
	}

	if contestName != "" && current.ContestName == "" {
		current.ContestName = contestName
	}
	current.ACProblemKeys[acceptedEvent.ProblemKey] = struct{}{}
	if acceptedEvent.AcceptedAt.Before(current.FirstACAt) {
		current.FirstACAt = acceptedEvent.AcceptedAt
	}
	if acceptedEvent.AcceptedAt.After(current.LastACAt) {
		current.LastACAt = acceptedEvent.AcceptedAt
	}

	aggregates[acceptedEvent.ContestID] = current
}

func mergeProblemFact(
	existing model.ProblemFact,
	incoming problemFactAggregate,
) problemFactAggregate {
	if existing.ProblemKey == "" {
		return incoming
	}

	if existing.ContestID != "" && incoming.ContestID == "" {
		incoming.ContestID = existing.ContestID
	}
	if existing.ProblemIndexOrTaskID != "" && incoming.ProblemIndex == "" {
		incoming.ProblemIndex = existing.ProblemIndexOrTaskID
	}
	if existing.ProblemName != "" && incoming.ProblemName == "" {
		incoming.ProblemName = existing.ProblemName
	}
	if existing.ProblemURL != "" && incoming.ProblemURL == "" {
		incoming.ProblemURL = existing.ProblemURL
	}

	if existing.FirstACAt.Before(incoming.FirstACAt) || existing.FirstACAt.Equal(incoming.FirstACAt) {
		incoming.FirstACAt = existing.FirstACAt
		incoming.FirstACSource = existing.FirstACSource
		incoming.FirstACSubmissionID = existing.FirstACSubmissionRef
		incoming.FirstACEventRawID = existing.FirstACEventRawID
	}

	if existing.LatestACAt.After(incoming.LatestACAt) {
		incoming.LatestACAt = existing.LatestACAt
	}

	return incoming
}

func mergeContestSummary(
	existing model.ContestACSummary,
	incoming contestSummaryAggregate,
) contestSummaryAggregate {
	if existing.ContestID == "" {
		return incoming
	}

	if existing.ContestName != "" && incoming.ContestName == "" {
		incoming.ContestName = existing.ContestName
	}

	if incoming.ACProblemKeys == nil {
		incoming.ACProblemKeys = make(map[string]struct{})
	}
	for _, problemKey := range existing.ACProblemKeys {
		incoming.ACProblemKeys[problemKey] = struct{}{}
	}

	if existing.FirstACAt != nil && existing.FirstACAt.Before(incoming.FirstACAt) {
		incoming.FirstACAt = *existing.FirstACAt
	}

	if existing.LastACAt != nil && existing.LastACAt.After(incoming.LastACAt) {
		incoming.LastACAt = *existing.LastACAt
	}

	return incoming
}

func (r *PlatformSyncRepository) insertProfileSnapshot(
	ctx context.Context,
	tx platformSyncTx,
	accountID int64,
	profile PlatformProfileSnapshotInput,
) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO platform_profile_snapshots (
  platform_account_id, source, display_name, rating, max_rating, profile_url, payload, fetched_at
)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), $7, $8)`,
		accountID,
		profile.Source,
		profile.DisplayName,
		profile.Rating,
		profile.MaxRating,
		profile.ProfileURL,
		profile.Payload,
		profile.FetchedAt.UTC(),
	)
	return err
}

func (r *CodeforcesSyncRepository) upsertContestHistoriesBatch(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	contests []CodeforcesContestHistoryInput,
) error {
	for start := 0; start < len(contests); start += codeforcesSyncBatchSize {
		end := minInt(start+codeforcesSyncBatchSize, len(contests))
		batch := contests[start:end]

		var builder strings.Builder
		builder.WriteString(`INSERT INTO platform_contest_histories (
  site_user_id, platform_account_id, platform, contest_id, contest_name, rank, old_rating, new_rating,
  rating_delta, participated_at, source, source_url, payload, fetched_at
)
VALUES `)

		args := make([]any, 0, len(batch)*14)
		for index, contest := range batch {
			if index > 0 {
				builder.WriteString(",")
			}

			argPos := len(args) + 1
			fmt.Fprintf(
				&builder,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6,
				argPos+7, argPos+8, argPos+9, argPos+10, argPos+11, argPos+12, argPos+13,
			)
			args = append(args,
				account.SiteUserID,
				account.ID,
				account.Platform,
				contest.ContestID,
				contest.ContestName,
				contest.Rank,
				contest.OldRating,
				contest.NewRating,
				contest.RatingDelta,
				contest.ParticipatedAt.UTC(),
				contest.Source,
				nullableString(contest.SourceURL),
				contest.Payload,
				contest.FetchedAt.UTC(),
			)
		}

		builder.WriteString(`
ON CONFLICT (platform_account_id, platform, contest_id) DO UPDATE
SET contest_name = EXCLUDED.contest_name,
    rank = EXCLUDED.rank,
    old_rating = EXCLUDED.old_rating,
    new_rating = EXCLUDED.new_rating,
    rating_delta = EXCLUDED.rating_delta,
    participated_at = EXCLUDED.participated_at,
    source = EXCLUDED.source,
    source_url = EXCLUDED.source_url,
    payload = EXCLUDED.payload,
    fetched_at = EXCLUDED.fetched_at,
    updated_at = NOW()`)

		if _, err := tx.Exec(ctx, builder.String(), args...); err != nil {
			return err
		}
	}

	return nil
}

func (r *PlatformSyncRepository) insertAcceptedEventRawBatch(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	acceptedEvents []PlatformAcceptedEventInput,
) ([]insertedAcceptedEventRaw, error) {
	inserted := make([]insertedAcceptedEventRaw, 0, len(acceptedEvents))
	for start := 0; start < len(acceptedEvents); start += codeforcesSyncBatchSize {
		end := minInt(start+codeforcesSyncBatchSize, len(acceptedEvents))
		batch := acceptedEvents[start:end]

		var builder strings.Builder
		builder.WriteString(`INSERT INTO accepted_event_raw (
  site_user_id, platform_account_id, platform, handle, problem_key, contest_id, problem_index_or_task_id,
  problem_name, problem_url, accepted_at, submission_id_or_ref, source, source_url, payload, fetched_at
)
VALUES `)

		args := make([]any, 0, len(batch)*15)
		for index, acceptedEvent := range batch {
			if index > 0 {
				builder.WriteString(",")
			}

			argPos := len(args) + 1
			fmt.Fprintf(
				&builder,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6, argPos+7,
				argPos+8, argPos+9, argPos+10, argPos+11, argPos+12, argPos+13, argPos+14,
			)
			args = append(args,
				account.SiteUserID,
				account.ID,
				account.Platform,
				acceptedEvent.Handle,
				acceptedEvent.ProblemKey,
				nullableString(acceptedEvent.ContestID),
				nullableString(acceptedEvent.ProblemIndex),
				nullableString(acceptedEvent.ProblemName),
				nullableString(acceptedEvent.ProblemURL),
				acceptedEvent.AcceptedAt.UTC(),
				nullableString(acceptedEvent.SubmissionID),
				acceptedEvent.Source,
				nullableString(acceptedEvent.SourceURL),
				acceptedEvent.Payload,
				acceptedEvent.FetchedAt.UTC(),
			)
		}

		builder.WriteString(`
ON CONFLICT (platform, handle, source, submission_id_or_ref) WHERE submission_id_or_ref IS NOT NULL DO UPDATE
SET source_url = EXCLUDED.source_url,
    payload = EXCLUDED.payload,
    fetched_at = EXCLUDED.fetched_at
RETURNING id, problem_key, accepted_at, COALESCE(submission_id_or_ref, '')`)

		rows, err := tx.Query(ctx, builder.String(), args...)
		if err != nil {
			return nil, err
		}

		batchInserted, err := collectInsertedAcceptedEventRaw(rows)
		rows.Close()
		if err != nil {
			return nil, err
		}

		inserted = append(inserted, batchInserted...)
	}

	return inserted, nil
}

func (r *PlatformSyncRepository) loadExistingProblemFactsByKeys(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	problemKeys []string,
) (map[string]model.ProblemFact, error) {
	if len(problemKeys) == 0 {
		return map[string]model.ProblemFact{}, nil
	}

	rows, err := tx.Query(
		ctx,
		`SELECT id, site_user_id, platform, problem_key, COALESCE(contest_id, ''),
       COALESCE(problem_index_or_task_id, ''), COALESCE(problem_name, ''), COALESCE(problem_url, ''),
       first_ac_at, first_ac_source, COALESCE(first_ac_submission_ref, ''), first_ac_event_raw_id,
       latest_ac_at, clist_problem_id, clist_contest_id, clist_rating, created_at, updated_at
FROM problem_facts
WHERE site_user_id = $1
  AND platform = $2
  AND problem_key = ANY($3)`,
		account.SiteUserID,
		account.Platform,
		problemKeys,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := collectProblemFacts(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]model.ProblemFact, len(items))
	for _, item := range items {
		result[item.ProblemKey] = item
	}

	return result, nil
}

func (r *PlatformSyncRepository) upsertProblemFactsBatch(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	problemFacts []problemFactAggregate,
) error {
	for start := 0; start < len(problemFacts); start += codeforcesSyncBatchSize {
		end := minInt(start+codeforcesSyncBatchSize, len(problemFacts))
		batch := problemFacts[start:end]

		var builder strings.Builder
		builder.WriteString(`INSERT INTO problem_facts (
  site_user_id, platform, problem_key, contest_id, problem_index_or_task_id, problem_name, problem_url,
  first_ac_at, first_ac_source, first_ac_submission_ref, first_ac_event_raw_id, latest_ac_at
)
VALUES `)

		args := make([]any, 0, len(batch)*12)
		for index, problemFact := range batch {
			if index > 0 {
				builder.WriteString(",")
			}

			argPos := len(args) + 1
			fmt.Fprintf(
				&builder,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5,
				argPos+6, argPos+7, argPos+8, argPos+9, argPos+10, argPos+11,
			)
			args = append(args,
				account.SiteUserID,
				account.Platform,
				problemFact.ProblemKey,
				nullableString(problemFact.ContestID),
				nullableString(problemFact.ProblemIndex),
				nullableString(problemFact.ProblemName),
				nullableString(problemFact.ProblemURL),
				problemFact.FirstACAt.UTC(),
				problemFact.FirstACSource,
				nullableString(problemFact.FirstACSubmissionID),
				problemFact.FirstACEventRawID,
				problemFact.LatestACAt.UTC(),
			)
		}

		builder.WriteString(`
ON CONFLICT (site_user_id, platform, problem_key) DO UPDATE
SET contest_id = EXCLUDED.contest_id,
    problem_index_or_task_id = EXCLUDED.problem_index_or_task_id,
    problem_name = EXCLUDED.problem_name,
    problem_url = EXCLUDED.problem_url,
    first_ac_at = EXCLUDED.first_ac_at,
    first_ac_source = EXCLUDED.first_ac_source,
    first_ac_submission_ref = EXCLUDED.first_ac_submission_ref,
    first_ac_event_raw_id = EXCLUDED.first_ac_event_raw_id,
    latest_ac_at = EXCLUDED.latest_ac_at,
    updated_at = NOW()`)

		if _, err := tx.Exec(ctx, builder.String(), args...); err != nil {
			return err
		}
	}

	return nil
}

func (r *PlatformSyncRepository) loadExistingContestSummariesByIDs(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	contestIDs []string,
) (map[string]model.ContestACSummary, error) {
	if len(contestIDs) == 0 {
		return map[string]model.ContestACSummary{}, nil
	}

	rows, err := tx.Query(
		ctx,
		`SELECT id, site_user_id, platform, contest_id, COALESCE(contest_name, ''), ac_problem_keys,
       ac_count, first_ac_at, last_ac_at, created_at, updated_at
FROM contest_ac_summaries
WHERE site_user_id = $1
  AND platform = $2
  AND contest_id = ANY($3)`,
		account.SiteUserID,
		account.Platform,
		contestIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := collectContestACSummaries(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]model.ContestACSummary, len(items))
	for _, item := range items {
		result[item.ContestID] = item
	}

	return result, nil
}

func (r *PlatformSyncRepository) upsertContestSummariesBatch(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	contestSummaries []contestSummaryAggregate,
) error {
	for start := 0; start < len(contestSummaries); start += codeforcesSyncBatchSize {
		end := minInt(start+codeforcesSyncBatchSize, len(contestSummaries))
		batch := contestSummaries[start:end]

		var builder strings.Builder
		builder.WriteString(`INSERT INTO contest_ac_summaries (
  site_user_id, platform, contest_id, contest_name, ac_problem_keys, ac_count, first_ac_at, last_ac_at
)
VALUES `)

		args := make([]any, 0, len(batch)*8)
		for index, contestSummary := range batch {
			if index > 0 {
				builder.WriteString(",")
			}

			problemKeys := sortedProblemKeys(contestSummary.ACProblemKeys)
			argPos := len(args) + 1
			fmt.Fprintf(
				&builder,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6, argPos+7,
			)
			args = append(args,
				account.SiteUserID,
				account.Platform,
				contestSummary.ContestID,
				nullableString(contestSummary.ContestName),
				problemKeys,
				len(problemKeys),
				contestSummary.FirstACAt.UTC(),
				contestSummary.LastACAt.UTC(),
			)
		}

		builder.WriteString(`
ON CONFLICT (site_user_id, platform, contest_id) DO UPDATE
SET contest_name = EXCLUDED.contest_name,
    ac_problem_keys = EXCLUDED.ac_problem_keys,
    ac_count = EXCLUDED.ac_count,
    first_ac_at = EXCLUDED.first_ac_at,
    last_ac_at = EXCLUDED.last_ac_at,
    updated_at = NOW()`)

		if _, err := tx.Exec(ctx, builder.String(), args...); err != nil {
			return err
		}
	}

	return nil
}

type platformProfileSnapshotScanner interface {
	Scan(dest ...any) error
}

func collectInsertedAcceptedEventRaw(rows pgx.Rows) ([]insertedAcceptedEventRaw, error) {
	items := make([]insertedAcceptedEventRaw, 0)
	for rows.Next() {
		var item insertedAcceptedEventRaw
		if err := rows.Scan(&item.ID, &item.ProblemKey, &item.AcceptedAt, &item.SubmissionID); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func scanPlatformProfileSnapshot(scanner platformProfileSnapshotScanner) (model.PlatformProfileSnapshot, error) {
	var snapshot model.PlatformProfileSnapshot
	var rating *int
	var maxRating *int
	err := scanner.Scan(
		&snapshot.ID,
		&snapshot.PlatformAccountID,
		&snapshot.Source,
		&snapshot.DisplayName,
		&rating,
		&maxRating,
		&snapshot.ProfileURL,
		&snapshot.RawPayloadRef,
		&snapshot.FetchedAt,
		&snapshot.CreatedAt,
	)
	if err != nil {
		return model.PlatformProfileSnapshot{}, err
	}

	snapshot.Rating = rating
	snapshot.MaxRating = maxRating
	return snapshot, nil
}

type platformContestHistoryScanner interface {
	Scan(dest ...any) error
}

func collectPlatformContestHistories(rows pgx.Rows) ([]model.PlatformContestHistory, error) {
	items := make([]model.PlatformContestHistory, 0)
	for rows.Next() {
		item, err := scanPlatformContestHistory(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func scanPlatformContestHistory(scanner platformContestHistoryScanner) (model.PlatformContestHistory, error) {
	var history model.PlatformContestHistory
	var platformAccountID *int64
	var rank *int
	var oldRating *int
	var newRating *int
	var ratingDelta *int
	err := scanner.Scan(
		&history.ID,
		&history.SiteUserID,
		&platformAccountID,
		&history.Platform,
		&history.ContestID,
		&history.ContestName,
		&rank,
		&oldRating,
		&newRating,
		&ratingDelta,
		&history.ParticipatedAt,
		&history.Source,
		&history.SourceURL,
		&history.RawPayloadRef,
		&history.FetchedAt,
		&history.CreatedAt,
		&history.UpdatedAt,
	)
	if err != nil {
		return model.PlatformContestHistory{}, err
	}

	history.PlatformAccountID = platformAccountID
	history.Rank = rank
	history.OldRating = oldRating
	history.NewRating = newRating
	history.RatingDelta = ratingDelta
	return history, nil
}

type problemFactScanner interface {
	Scan(dest ...any) error
}

func collectProblemFacts(rows pgx.Rows) ([]model.ProblemFact, error) {
	items := make([]model.ProblemFact, 0)
	for rows.Next() {
		item, err := scanProblemFact(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func scanProblemFact(scanner problemFactScanner) (model.ProblemFact, error) {
	var fact model.ProblemFact
	var firstACEventRawID *int64
	var clistProblemID *int64
	var clistContestID *int64
	var clistRating *int
	err := scanner.Scan(
		&fact.ID,
		&fact.SiteUserID,
		&fact.Platform,
		&fact.ProblemKey,
		&fact.ContestID,
		&fact.ProblemIndexOrTaskID,
		&fact.ProblemName,
		&fact.ProblemURL,
		&fact.FirstACAt,
		&fact.FirstACSource,
		&fact.FirstACSubmissionRef,
		&firstACEventRawID,
		&fact.LatestACAt,
		&clistProblemID,
		&clistContestID,
		&clistRating,
		&fact.CreatedAt,
		&fact.UpdatedAt,
	)
	if err != nil {
		return model.ProblemFact{}, err
	}

	fact.FirstACEventRawID = firstACEventRawID
	fact.ClistProblemID = clistProblemID
	fact.ClistContestID = clistContestID
	fact.ClistRating = clistRating
	return fact, nil
}

type contestACSummaryScanner interface {
	Scan(dest ...any) error
}

func collectContestACSummaries(rows pgx.Rows) ([]model.ContestACSummary, error) {
	items := make([]model.ContestACSummary, 0)
	for rows.Next() {
		item, err := scanContestACSummary(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func scanContestACSummary(scanner contestACSummaryScanner) (model.ContestACSummary, error) {
	var summary model.ContestACSummary
	var firstACAt *time.Time
	var lastACAt *time.Time
	err := scanner.Scan(
		&summary.ID,
		&summary.SiteUserID,
		&summary.Platform,
		&summary.ContestID,
		&summary.ContestName,
		&summary.ACProblemKeys,
		&summary.ACCount,
		&firstACAt,
		&lastACAt,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err != nil {
		return model.ContestACSummary{}, err
	}

	summary.FirstACAt = firstACAt
	summary.LastACAt = lastACAt
	return summary, nil
}

func acceptedEventIdentity(
	submissionID string,
	problemKey string,
	acceptedAt time.Time,
) string {
	if submissionID != "" {
		return submissionID
	}

	return problemKey + "|" + acceptedAt.UTC().Format(time.RFC3339Nano)
}

func sortedProblemKeys(problemKeys map[string]struct{}) []string {
	result := make([]string, 0, len(problemKeys))
	for problemKey := range problemKeys {
		result = append(result, problemKey)
	}
	sort.Strings(result)
	return result
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}

	return right
}
