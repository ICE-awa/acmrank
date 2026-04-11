package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type LuoguProfileSnapshotInput = PlatformProfileSnapshotInput

type LuoguAcceptedProblemInput struct {
	Handle      string
	ProblemKey  string
	ProblemID   string
	ProblemName string
	ProblemURL  string
	AcceptedAt  time.Time
	Source      string
	SourceURL   string
	Payload     []byte
	FetchedAt   time.Time
}

type SaveLuoguSyncParams struct {
	Account          model.PlatformAccount
	SyncedAt         time.Time
	Profile          LuoguProfileSnapshotInput
	AcceptedProblems []LuoguAcceptedProblemInput
}

type LuoguSyncRepository struct {
	*PlatformSyncRepository
}

func NewLuoguSyncRepository(db platformSyncRepositoryDB) *LuoguSyncRepository {
	return &LuoguSyncRepository{
		PlatformSyncRepository: NewPlatformSyncRepository(db),
	}
}

func (r *LuoguSyncRepository) SaveSync(
	ctx context.Context,
	params SaveLuoguSyncParams,
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

	acceptedEvents := toLuoguAcceptedEventInputs(params.AcceptedProblems)
	sort.Slice(acceptedEvents, func(i, j int) bool {
		if acceptedEvents[i].ProblemKey == acceptedEvents[j].ProblemKey {
			return acceptedEvents[i].SubmissionID < acceptedEvents[j].SubmissionID
		}
		return acceptedEvents[i].ProblemKey < acceptedEvents[j].ProblemKey
	})

	insertedAcceptedEvents, err := r.insertAcceptedEventRawBatch(ctx, tx, params.Account, acceptedEvents)
	if err != nil {
		return err
	}

	rawIDByAcceptedEvent := make(map[string]int64, len(insertedAcceptedEvents))
	for _, insertedAcceptedEvent := range insertedAcceptedEvents {
		rawIDByAcceptedEvent[insertedAcceptedEvent.ProblemKey] = insertedAcceptedEvent.ID
	}

	problemFacts := make(map[string]problemFactAggregate, len(acceptedEvents))
	for _, acceptedEvent := range acceptedEvents {
		rawID, ok := rawIDByAcceptedEvent[acceptedEvent.ProblemKey]
		if !ok {
			return fmt.Errorf("accepted event raw id missing for luogu problem %q", acceptedEvent.ProblemKey)
		}

		aggregateProblemFact(problemFacts, acceptedEvent, rawID)
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
		mergedProblemFacts = append(mergedProblemFacts, mergeLuoguProblemFact(
			existingProblemFacts[problemKey],
			problemFacts[problemKey],
		))
	}
	if err := r.upsertProblemFactsBatch(ctx, tx, params.Account, mergedProblemFacts); err != nil {
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

func mergeLuoguProblemFact(
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

	incoming.FirstACAt = existing.FirstACAt
	incoming.FirstACSource = existing.FirstACSource
	incoming.FirstACSubmissionID = existing.FirstACSubmissionRef
	incoming.FirstACEventRawID = copyOptionalInt64(existing.FirstACEventRawID)
	incoming.LatestACAt = existing.LatestACAt

	return incoming
}

func toLuoguAcceptedEventInputs(
	problems []LuoguAcceptedProblemInput,
) []PlatformAcceptedEventInput {
	result := make([]PlatformAcceptedEventInput, 0, len(problems))
	for _, problem := range problems {
		result = append(result, PlatformAcceptedEventInput{
			Handle:       problem.Handle,
			ProblemKey:   problem.ProblemKey,
			ProblemIndex: problem.ProblemID,
			ProblemName:  problem.ProblemName,
			ProblemURL:   problem.ProblemURL,
			AcceptedAt:   problem.AcceptedAt,
			SubmissionID: problem.ProblemID,
			Source:       problem.Source,
			SourceURL:    problem.SourceURL,
			Payload:      problem.Payload,
			FetchedAt:    problem.FetchedAt,
		})
	}

	return result
}
