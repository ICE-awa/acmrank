package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrPlatformSyncDataNotFound = errors.New("platform sync data not found")

type codeforcesSyncRepositoryDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type codeforcesSyncTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type pgxCodeforcesSyncTx struct {
	tx pgx.Tx
}

func (t pgxCodeforcesSyncTx) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t pgxCodeforcesSyncTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t pgxCodeforcesSyncTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxCodeforcesSyncTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type CodeforcesProfileSnapshotInput struct {
	DisplayName string
	Rating      *int
	MaxRating   *int
	ProfileURL  string
	Source      string
	Payload     []byte
	FetchedAt   time.Time
}

type CodeforcesAcceptedEventInput struct {
	Handle       string
	ProblemKey   string
	ContestID    string
	ProblemIndex string
	ProblemName  string
	ProblemURL   string
	AcceptedAt   time.Time
	SubmissionID string
	Source       string
	SourceURL    string
	Payload      []byte
	FetchedAt    time.Time
}

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
	Profile          CodeforcesProfileSnapshotInput
	AcceptedEvents   []CodeforcesAcceptedEventInput
	ContestHistories []CodeforcesContestHistoryInput
}

type ListCodeforcesSyncFilter struct {
	Limit  int
	Offset int
}

type CodeforcesSyncRepository struct {
	db      codeforcesSyncRepositoryDB
	beginTx func(context.Context) (codeforcesSyncTx, error)
}

func NewCodeforcesSyncRepository(db codeforcesSyncRepositoryDB) *CodeforcesSyncRepository {
	return &CodeforcesSyncRepository{
		db: db,
		beginTx: func(ctx context.Context) (codeforcesSyncTx, error) {
			tx, err := db.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return nil, err
			}

			return pgxCodeforcesSyncTx{tx: tx}, nil
		},
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
	for _, contest := range params.ContestHistories {
		if err := r.upsertContestHistory(ctx, tx, params.Account, contest); err != nil {
			return err
		}

		contestNameByID[contest.ContestID] = contest.ContestName
	}

	sortedAccepted := append([]CodeforcesAcceptedEventInput(nil), params.AcceptedEvents...)
	sort.Slice(sortedAccepted, func(i, j int) bool {
		if sortedAccepted[i].AcceptedAt.Equal(sortedAccepted[j].AcceptedAt) {
			return sortedAccepted[i].SubmissionID < sortedAccepted[j].SubmissionID
		}

		return sortedAccepted[i].AcceptedAt.Before(sortedAccepted[j].AcceptedAt)
	})

	problemFacts := make(map[string]problemFactAggregate)
	contestSummaries := make(map[string]contestSummaryAggregate)
	for _, acceptedEvent := range sortedAccepted {
		rawID, err := r.upsertAcceptedEventRaw(ctx, tx, params.Account, acceptedEvent)
		if err != nil {
			return err
		}

		aggregateProblemFact(problemFacts, acceptedEvent, rawID)
		aggregateContestSummary(contestSummaries, acceptedEvent, contestNameByID[acceptedEvent.ContestID])
	}

	problemKeys := make([]string, 0, len(problemFacts))
	for problemKey := range problemFacts {
		problemKeys = append(problemKeys, problemKey)
	}
	sort.Strings(problemKeys)

	for _, problemKey := range problemKeys {
		if err := r.upsertProblemFact(ctx, tx, params.Account, problemFacts[problemKey]); err != nil {
			return err
		}
	}

	contestIDs := make([]string, 0, len(contestSummaries))
	for contestID := range contestSummaries {
		contestIDs = append(contestIDs, contestID)
	}
	sort.Strings(contestIDs)

	for _, contestID := range contestIDs {
		if err := r.upsertContestSummary(ctx, tx, params.Account, contestSummaries[contestID]); err != nil {
			return err
		}
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

func (r *CodeforcesSyncRepository) GetLatestProfileSnapshot(
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
	filter ListCodeforcesSyncFilter,
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

func (r *CodeforcesSyncRepository) ListProblemFactsByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter ListCodeforcesSyncFilter,
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

func (r *CodeforcesSyncRepository) ListContestSummariesByUserIDAndPlatform(
	ctx context.Context,
	siteUserID int64,
	platform model.Platform,
	filter ListCodeforcesSyncFilter,
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
	FirstACEventRawID   int64
	LatestACAt          time.Time
}

type contestSummaryAggregate struct {
	ContestID     string
	ContestName   string
	ACProblemKeys map[string]struct{}
	FirstACAt     time.Time
	LastACAt      time.Time
}

func aggregateProblemFact(
	aggregates map[string]problemFactAggregate,
	acceptedEvent CodeforcesAcceptedEventInput,
	rawID int64,
) {
	current, exists := aggregates[acceptedEvent.ProblemKey]
	if !exists {
		aggregates[acceptedEvent.ProblemKey] = problemFactAggregate{
			ProblemKey:          acceptedEvent.ProblemKey,
			ContestID:           acceptedEvent.ContestID,
			ProblemIndex:        acceptedEvent.ProblemIndex,
			ProblemName:         acceptedEvent.ProblemName,
			ProblemURL:          acceptedEvent.ProblemURL,
			FirstACAt:           acceptedEvent.AcceptedAt,
			FirstACSource:       acceptedEvent.Source,
			FirstACSubmissionID: acceptedEvent.SubmissionID,
			FirstACEventRawID:   rawID,
			LatestACAt:          acceptedEvent.AcceptedAt,
		}
		return
	}

	if acceptedEvent.AcceptedAt.Before(current.FirstACAt) {
		current.FirstACAt = acceptedEvent.AcceptedAt
		current.FirstACSource = acceptedEvent.Source
		current.FirstACSubmissionID = acceptedEvent.SubmissionID
		current.FirstACEventRawID = rawID
	}
	if acceptedEvent.AcceptedAt.After(current.LatestACAt) {
		current.LatestACAt = acceptedEvent.AcceptedAt
	}

	aggregates[acceptedEvent.ProblemKey] = current
}

func aggregateContestSummary(
	aggregates map[string]contestSummaryAggregate,
	acceptedEvent CodeforcesAcceptedEventInput,
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

func (r *CodeforcesSyncRepository) insertProfileSnapshot(
	ctx context.Context,
	tx codeforcesSyncTx,
	accountID int64,
	profile CodeforcesProfileSnapshotInput,
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

func (r *CodeforcesSyncRepository) upsertContestHistory(
	ctx context.Context,
	tx codeforcesSyncTx,
	account model.PlatformAccount,
	contest CodeforcesContestHistoryInput,
) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO platform_contest_histories (
  site_user_id, platform_account_id, platform, contest_id, contest_name, rank, old_rating, new_rating,
  rating_delta, participated_at, source, source_url, payload, fetched_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, ''), $13, $14)
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
    updated_at = NOW()`,
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
		contest.SourceURL,
		contest.Payload,
		contest.FetchedAt.UTC(),
	)
	return err
}

func (r *CodeforcesSyncRepository) upsertAcceptedEventRaw(
	ctx context.Context,
	tx codeforcesSyncTx,
	account model.PlatformAccount,
	acceptedEvent CodeforcesAcceptedEventInput,
) (int64, error) {
	var rawID int64
	err := tx.QueryRow(
		ctx,
		`INSERT INTO accepted_event_raw (
  site_user_id, platform_account_id, platform, handle, problem_key, contest_id, problem_index_or_task_id,
  problem_name, problem_url, accepted_at, submission_id_or_ref, source, source_url, payload, fetched_at
)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12, NULLIF($13, ''), $14, $15)
ON CONFLICT (platform, handle, source, submission_id_or_ref) WHERE submission_id_or_ref IS NOT NULL DO UPDATE
SET source_url = EXCLUDED.source_url,
    payload = EXCLUDED.payload,
    fetched_at = EXCLUDED.fetched_at
RETURNING id`,
		account.SiteUserID,
		account.ID,
		account.Platform,
		acceptedEvent.Handle,
		acceptedEvent.ProblemKey,
		acceptedEvent.ContestID,
		acceptedEvent.ProblemIndex,
		acceptedEvent.ProblemName,
		acceptedEvent.ProblemURL,
		acceptedEvent.AcceptedAt.UTC(),
		acceptedEvent.SubmissionID,
		acceptedEvent.Source,
		acceptedEvent.SourceURL,
		acceptedEvent.Payload,
		acceptedEvent.FetchedAt.UTC(),
	).Scan(&rawID)
	if err != nil {
		return 0, err
	}

	return rawID, nil
}

func (r *CodeforcesSyncRepository) upsertProblemFact(
	ctx context.Context,
	tx codeforcesSyncTx,
	account model.PlatformAccount,
	problemFact problemFactAggregate,
) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO problem_facts (
  site_user_id, platform, problem_key, contest_id, problem_index_or_task_id, problem_name, problem_url,
  first_ac_at, first_ac_source, first_ac_submission_ref, first_ac_event_raw_id, latest_ac_at
)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9, NULLIF($10, ''), $11, $12)
ON CONFLICT (site_user_id, platform, problem_key) DO UPDATE
SET contest_id = COALESCE(problem_facts.contest_id, EXCLUDED.contest_id),
    problem_index_or_task_id = COALESCE(problem_facts.problem_index_or_task_id, EXCLUDED.problem_index_or_task_id),
    problem_name = COALESCE(problem_facts.problem_name, EXCLUDED.problem_name),
    problem_url = COALESCE(problem_facts.problem_url, EXCLUDED.problem_url),
    first_ac_source = CASE
        WHEN EXCLUDED.first_ac_at < problem_facts.first_ac_at THEN EXCLUDED.first_ac_source
        ELSE problem_facts.first_ac_source
    END,
    first_ac_submission_ref = CASE
        WHEN EXCLUDED.first_ac_at < problem_facts.first_ac_at THEN EXCLUDED.first_ac_submission_ref
        ELSE problem_facts.first_ac_submission_ref
    END,
    first_ac_event_raw_id = CASE
        WHEN EXCLUDED.first_ac_at < problem_facts.first_ac_at THEN EXCLUDED.first_ac_event_raw_id
        ELSE problem_facts.first_ac_event_raw_id
    END,
    first_ac_at = LEAST(problem_facts.first_ac_at, EXCLUDED.first_ac_at),
    latest_ac_at = GREATEST(problem_facts.latest_ac_at, EXCLUDED.latest_ac_at),
    updated_at = NOW()`,
		account.SiteUserID,
		account.Platform,
		problemFact.ProblemKey,
		problemFact.ContestID,
		problemFact.ProblemIndex,
		problemFact.ProblemName,
		problemFact.ProblemURL,
		problemFact.FirstACAt.UTC(),
		problemFact.FirstACSource,
		problemFact.FirstACSubmissionID,
		problemFact.FirstACEventRawID,
		problemFact.LatestACAt.UTC(),
	)
	return err
}

func (r *CodeforcesSyncRepository) upsertContestSummary(
	ctx context.Context,
	tx codeforcesSyncTx,
	account model.PlatformAccount,
	contestSummary contestSummaryAggregate,
) error {
	problemKeys := make([]string, 0, len(contestSummary.ACProblemKeys))
	for problemKey := range contestSummary.ACProblemKeys {
		problemKeys = append(problemKeys, problemKey)
	}
	sort.Strings(problemKeys)

	_, err := tx.Exec(
		ctx,
		`INSERT INTO contest_ac_summaries (
  site_user_id, platform, contest_id, contest_name, ac_problem_keys, ac_count, first_ac_at, last_ac_at
)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8)
ON CONFLICT (site_user_id, platform, contest_id) DO UPDATE
SET contest_name = COALESCE(contest_ac_summaries.contest_name, EXCLUDED.contest_name),
    ac_problem_keys = ARRAY(
      SELECT DISTINCT problem_key
      FROM unnest(contest_ac_summaries.ac_problem_keys || EXCLUDED.ac_problem_keys) AS problem_key
      ORDER BY problem_key
    ),
    ac_count = (
      SELECT COUNT(DISTINCT problem_key)
      FROM unnest(contest_ac_summaries.ac_problem_keys || EXCLUDED.ac_problem_keys) AS problem_key
    ),
    first_ac_at = CASE
        WHEN contest_ac_summaries.first_ac_at IS NULL THEN EXCLUDED.first_ac_at
        ELSE LEAST(contest_ac_summaries.first_ac_at, EXCLUDED.first_ac_at)
    END,
    last_ac_at = CASE
        WHEN contest_ac_summaries.last_ac_at IS NULL THEN EXCLUDED.last_ac_at
        ELSE GREATEST(contest_ac_summaries.last_ac_at, EXCLUDED.last_ac_at)
    END,
    updated_at = NOW()`,
		account.SiteUserID,
		account.Platform,
		contestSummary.ContestID,
		contestSummary.ContestName,
		problemKeys,
		len(problemKeys),
		contestSummary.FirstACAt.UTC(),
		contestSummary.LastACAt.UTC(),
	)
	return err
}

type platformProfileSnapshotScanner interface {
	Scan(dest ...any) error
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
