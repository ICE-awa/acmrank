package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type AtCoderContestHistoryInput struct {
	ContestID      string
	ContestName    string
	Rank           *int
	OldRating      *int
	NewRating      *int
	RatingDelta    *int
	ParticipatedAt time.Time
	Source         string
	SourceURL      string
	Payload        []byte
	FetchedAt      time.Time
}

type SaveAtCoderSyncParams struct {
	Account          model.PlatformAccount
	SyncedAt         time.Time
	Profile          PlatformProfileSnapshotInput
	AcceptedEvents   []PlatformAcceptedEventInput
	ContestHistories []AtCoderContestHistoryInput
}

type AtCoderSyncRepository struct {
	*PlatformSyncRepository
}

func NewAtCoderSyncRepository(db platformSyncRepositoryDB) *AtCoderSyncRepository {
	return &AtCoderSyncRepository{
		PlatformSyncRepository: NewPlatformSyncRepository(db),
	}
}

func (r *AtCoderSyncRepository) SaveSync(
	ctx context.Context,
	params SaveAtCoderSyncParams,
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

func (r *AtCoderSyncRepository) ListContestHistoriesByAccountID(
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

func (r *AtCoderSyncRepository) upsertContestHistoriesBatch(
	ctx context.Context,
	tx platformSyncTx,
	account model.PlatformAccount,
	contests []AtCoderContestHistoryInput,
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
