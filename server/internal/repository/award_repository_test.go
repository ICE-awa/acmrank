package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubAwardDB struct {
	queryFn func(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *stubAwardDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return s.queryFn(ctx, sql, args...)
}

func (s *stubAwardDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 0"), nil
}

func (s *stubAwardDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, nil
}

type stubAwardTx struct {
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
	commitFn   func(context.Context) error
	rollbackFn func(context.Context) error
}

func (s stubAwardTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return s.execFn(ctx, sql, args...)
}

func (s stubAwardTx) Commit(ctx context.Context) error {
	if s.commitFn != nil {
		return s.commitFn(ctx)
	}

	return nil
}

func (s stubAwardTx) Rollback(ctx context.Context) error {
	if s.rollbackFn != nil {
		return s.rollbackFn(ctx)
	}

	return nil
}

func TestAwardRepositoryReplaceAutoSyncByUserIDPreservesManualSlot(t *testing.T) {
	t.Parallel()

	awardDate := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC)
	deleteCalled := false
	insertCalls := 0

	repository := NewAwardRepository(&stubAwardDB{
		queryFn: func(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil },
	})
	repository.beginTx = func(context.Context) (awardTx, error) {
		return stubAwardTx{
			execFn: func(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
				switch {
				case strings.Contains(query, "DELETE FROM award_records"):
					deleteCalled = true
					if len(args) != 2 || args[0] != int64(7) || args[1] != "icpc" {
						t.Fatalf("DELETE args = %#v", args)
					}
				case strings.Contains(query, "INSERT INTO award_records"):
					insertCalls++
					if !strings.Contains(query, "$10") {
						t.Fatalf("expected batched INSERT query, got %q", query)
					}
					if len(args) != 18 || args[2] != "ICPC Asia Regional 2025" || args[3] != "Gold Medal" {
						t.Fatalf("INSERT args = %#v", args)
					}
					if args[11] != "ICPC EC Final 2024" || args[12] != "Silver Medal" {
						t.Fatalf("INSERT args = %#v", args)
					}
				default:
					t.Fatalf("unexpected query = %q", query)
				}

				return pgconn.NewCommandTag("INSERT 0 1"), nil
			},
		}, nil
	}

	err := repository.ReplaceAutoSyncByUserID(context.Background(), ReplaceAwardRecordsParams{
		SiteUserID: 7,
		Platform:   "icpc",
		Records: []AwardRecordInput{
			{
				ContestName: "ICPC Asia Regional 2025",
				AwardName:   "Gold Medal",
				RankText:    "Rank 3",
				AwardDate:   awardDate,
				Source:      "icpc_awards_feed",
				SourceURL:   "https://board.example.test/regional-2025",
			},
			{
				ContestName: "ICPC EC Final 2024",
				AwardName:   "Silver Medal",
				RankText:    "Rank 5",
				AwardDate:   time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC),
				Source:      "icpc_awards_feed",
				SourceURL:   "https://board.example.test/final-2024",
			},
		},
	})
	if err != nil {
		t.Fatalf("ReplaceAutoSyncByUserID() error = %v", err)
	}

	if !deleteCalled || insertCalls != 1 {
		t.Fatalf("deleteCalled=%v insertCalls=%d, want true/1", deleteCalled, insertCalls)
	}
}

func TestAwardRepositoryListByUserIDScansRecords(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	repository := NewAwardRepository(&stubAwardDB{
		queryFn: func(_ context.Context, query string, args ...any) (pgx.Rows, error) {
			if !strings.Contains(query, "FROM award_records") {
				t.Fatalf("unexpected query = %q", query)
			}

			return &stubCodeforcesSyncRows{
				scanFns: []func(dest ...any) error{
					func(dest ...any) error {
						*(dest[0].(*int64)) = 1
						*(dest[1].(*int64)) = 7
						*(dest[2].(*string)) = "icpc"
						*(dest[3].(*string)) = "ICPC EC Final 2024"
						*(dest[4].(*string)) = "Silver Medal"
						*(dest[5].(*string)) = "Rank 5"
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*string)) = "icpc_awards_feed"
						*(dest[8].(*string)) = "https://board.example.test/final-2024"
						*(dest[9].(*bool)) = false
						*(dest[10].(*string)) = ""
						*(dest[11].(*time.Time)) = now
						return nil
					},
				},
			}, nil
		},
	})

	records, err := repository.ListByUserID(context.Background(), 7, ListAwardRecordsFilter{
		Limit:  20,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}

	if len(records) != 1 || records[0].AwardName != "Silver Medal" {
		t.Fatalf("ListByUserID() records = %#v", records)
	}
}
