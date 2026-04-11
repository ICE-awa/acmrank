package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubCodeforcesSyncDB struct {
	queryFn    func(context.Context, string, ...any) (pgx.Rows, error)
	queryRowFn func(context.Context, string, ...any) pgx.Row
	beginTxFn  func(context.Context) (codeforcesSyncTx, error)
}

func (s *stubCodeforcesSyncDB) Query(
	ctx context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	return s.queryFn(ctx, query, args...)
}

func (s *stubCodeforcesSyncDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, query, args...)
}

func (s *stubCodeforcesSyncDB) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	if s.beginTxFn == nil {
		return nil, errors.New("unexpected BeginTx call")
	}

	return nil, errors.New("unused")
}

type stubCodeforcesSyncTx struct {
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
	queryFn    func(context.Context, string, ...any) (pgx.Rows, error)
	queryRowFn func(context.Context, string, ...any) pgx.Row
	commitFn   func(context.Context) error
	rollbackFn func(context.Context) error
}

func (s stubCodeforcesSyncTx) Exec(
	ctx context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	return s.execFn(ctx, query, args...)
}

func (s stubCodeforcesSyncTx) Query(
	ctx context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	return s.queryFn(ctx, query, args...)
}

func (s stubCodeforcesSyncTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, query, args...)
}

func (s stubCodeforcesSyncTx) Commit(ctx context.Context) error {
	if s.commitFn != nil {
		return s.commitFn(ctx)
	}

	return nil
}

func (s stubCodeforcesSyncTx) Rollback(ctx context.Context) error {
	if s.rollbackFn != nil {
		return s.rollbackFn(ctx)
	}

	return nil
}

type stubCodeforcesSyncRows struct {
	index   int
	scanFns []func(dest ...any) error
	err     error
}

func (r *stubCodeforcesSyncRows) Close()     {}
func (r *stubCodeforcesSyncRows) Err() error { return r.err }
func (r *stubCodeforcesSyncRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}
func (r *stubCodeforcesSyncRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *stubCodeforcesSyncRows) Next() bool                                   { return r.index < len(r.scanFns) }
func (r *stubCodeforcesSyncRows) Scan(dest ...any) error {
	scanFn := r.scanFns[r.index]
	r.index++
	return scanFn(dest...)
}
func (r *stubCodeforcesSyncRows) Values() ([]any, error) { return nil, nil }
func (r *stubCodeforcesSyncRows) RawValues() [][]byte    { return nil }
func (r *stubCodeforcesSyncRows) Conn() *pgx.Conn        { return nil }

func TestCodeforcesSyncRepositoryGetLatestProfileSnapshotMapsNotFound(t *testing.T) {
	t.Parallel()

	repository := NewCodeforcesSyncRepository(&stubCodeforcesSyncDB{
		queryFn: func(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil },
		queryRowFn: func(context.Context, string, ...any) pgx.Row {
			return stubRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := repository.GetLatestProfileSnapshot(context.Background(), 3)
	if !errors.Is(err, ErrPlatformSyncDataNotFound) {
		t.Fatalf("GetLatestProfileSnapshot() error = %v, want %v", err, ErrPlatformSyncDataNotFound)
	}
}

func TestCodeforcesSyncRepositoryListContestHistoriesAppliesPagination(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_600_000, 0).UTC()
	repository := NewCodeforcesSyncRepository(&stubCodeforcesSyncDB{
		queryFn: func(_ context.Context, query string, args ...any) (pgx.Rows, error) {
			if !strings.Contains(query, "WHERE platform_account_id = $1") || len(args) != 3 || args[1] != 25 || args[2] != 10 {
				t.Fatalf("ListContestHistoriesByAccountID() query=%q args=%#v", query, args)
			}

			return &stubCodeforcesSyncRows{
				scanFns: []func(dest ...any) error{
					func(dest ...any) error {
						assignPlatformContestHistoryRow(dest, model.PlatformContestHistory{
							ID:             1,
							SiteUserID:     7,
							Platform:       model.PlatformCodeforces,
							ContestID:      "1000",
							ContestName:    "Round 1000",
							ParticipatedAt: now,
							Source:         model.SyncSourceCodeforcesAPI,
							FetchedAt:      now,
							CreatedAt:      now,
							UpdatedAt:      now,
						}, int64Ptr(4), intPtr(5), intPtr(3400), intPtr(3500), intPtr(100))
						return nil
					},
				},
			}, nil
		},
		queryRowFn: func(context.Context, string, ...any) pgx.Row { return stubRow{} },
	})

	items, err := repository.ListContestHistoriesByAccountID(context.Background(), 4, ListCodeforcesSyncFilter{
		Limit:  25,
		Offset: 10,
	})
	if err != nil {
		t.Fatalf("ListContestHistoriesByAccountID() error = %v", err)
	}

	if len(items) != 1 || items[0].ContestName != "Round 1000" {
		t.Fatalf("ListContestHistoriesByAccountID() items = %+v", items)
	}
}

func TestCodeforcesSyncRepositorySaveSyncUpdatesLastSyncedAt(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_600_100, 0).UTC()
	var updatedLastSynced bool
	repository := NewCodeforcesSyncRepository(&stubCodeforcesSyncDB{
		queryFn:    func(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil },
		queryRowFn: func(context.Context, string, ...any) pgx.Row { return stubRow{} },
	})
	repository.beginTx = func(context.Context) (codeforcesSyncTx, error) {
		return stubCodeforcesSyncTx{
			execFn: func(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
				if strings.Contains(query, "UPDATE platform_accounts") {
					updatedLastSynced = true
					if len(args) != 2 || args[0] != int64(4) || args[1] != now {
						t.Fatalf("UPDATE platform_accounts args = %#v", args)
					}
				}

				return pgconn.NewCommandTag("INSERT 0 1"), nil
			},
			queryFn: func(_ context.Context, query string, args ...any) (pgx.Rows, error) {
				switch {
				case strings.Contains(query, "INSERT INTO accepted_event_raw"):
					return &stubCodeforcesSyncRows{
						scanFns: []func(dest ...any) error{
							func(dest ...any) error {
								*(dest[0].(*int64)) = 9
								*(dest[1].(*string)) = "CF-1000A"
								*(dest[2].(*time.Time)) = now
								*(dest[3].(*string)) = "1"
								return nil
							},
						},
					}, nil
				case strings.Contains(query, "FROM problem_facts"):
					return &stubCodeforcesSyncRows{}, nil
				case strings.Contains(query, "FROM contest_ac_summaries"):
					return &stubCodeforcesSyncRows{}, nil
				default:
					t.Fatalf("unexpected Query() query = %q", query)
					return nil, nil
				}
			},
			queryRowFn: func(context.Context, string, ...any) pgx.Row { return stubRow{} },
			commitFn:   func(context.Context) error { return nil },
		}, nil
	}

	err := repository.SaveSync(context.Background(), SaveCodeforcesSyncParams{
		Account: model.PlatformAccount{
			ID:         4,
			SiteUserID: 7,
			Platform:   model.PlatformCodeforces,
			Handle:     "tourist",
		},
		SyncedAt: now,
		Profile: CodeforcesProfileSnapshotInput{
			DisplayName: "tourist",
			Source:      model.SyncSourceCodeforcesAPI,
			FetchedAt:   now,
		},
		AcceptedEvents: []CodeforcesAcceptedEventInput{
			{
				Handle:       "tourist",
				ProblemKey:   "CF-1000A",
				ContestID:    "1000",
				ProblemIndex: "A",
				ProblemName:  "Problem A",
				ProblemURL:   "https://codeforces.com/contest/1000/problem/A",
				AcceptedAt:   now,
				SubmissionID: "1",
				Source:       model.SyncSourceCodeforcesAPI,
				SourceURL:    "https://codeforces.com/contest/1000/submission/1",
				FetchedAt:    now,
			},
		},
		ContestHistories: []CodeforcesContestHistoryInput{
			{
				ContestID:      "1000",
				ContestName:    "Round 1000",
				Rank:           5,
				OldRating:      3400,
				NewRating:      3500,
				RatingDelta:    100,
				ParticipatedAt: now,
				Source:         model.SyncSourceCodeforcesAPI,
				SourceURL:      "https://codeforces.com/contest/1000",
				FetchedAt:      now,
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveSync() error = %v", err)
	}

	if !updatedLastSynced {
		t.Fatal("SaveSync() expected platform_accounts.last_synced_at to be updated")
	}
}

func assignPlatformContestHistoryRow(
	dest []any,
	history model.PlatformContestHistory,
	platformAccountID *int64,
	rank *int,
	oldRating *int,
	newRating *int,
	ratingDelta *int,
) {
	*(dest[0].(*int64)) = history.ID
	*(dest[1].(*int64)) = history.SiteUserID
	*(dest[2].(**int64)) = platformAccountID
	*(dest[3].(*model.Platform)) = history.Platform
	*(dest[4].(*string)) = history.ContestID
	*(dest[5].(*string)) = history.ContestName
	*(dest[6].(**int)) = rank
	*(dest[7].(**int)) = oldRating
	*(dest[8].(**int)) = newRating
	*(dest[9].(**int)) = ratingDelta
	*(dest[10].(*time.Time)) = history.ParticipatedAt
	*(dest[11].(*string)) = history.Source
	*(dest[12].(*string)) = history.SourceURL
	*(dest[13].(*string)) = history.RawPayloadRef
	*(dest[14].(*time.Time)) = history.FetchedAt
	*(dest[15].(*time.Time)) = history.CreatedAt
	*(dest[16].(*time.Time)) = history.UpdatedAt
}

func int64Ptr(value int64) *int64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}
