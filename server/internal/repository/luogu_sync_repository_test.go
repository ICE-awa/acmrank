package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestLuoguSyncRepositorySaveSyncUpdatesLastSyncedAt(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_800_000, 0).UTC()
	var updatedLastSynced bool
	repository := NewLuoguSyncRepository(&stubCodeforcesSyncDB{
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
								*(dest[1].(*string)) = "P1001"
								*(dest[2].(*time.Time)) = now
								*(dest[3].(*string)) = "P1001"
								return nil
							},
						},
					}, nil
				case strings.Contains(query, "FROM problem_facts"):
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

	err := repository.SaveSync(context.Background(), SaveLuoguSyncParams{
		Account: model.PlatformAccount{
			ID:         4,
			SiteUserID: 7,
			Platform:   model.PlatformLuogu,
			Handle:     "qiaochu",
		},
		SyncedAt: now,
		Profile: LuoguProfileSnapshotInput{
			DisplayName: "qiaochu",
			Source:      model.SyncSourceLuoguUserInfo,
			FetchedAt:   now,
		},
		AcceptedProblems: []LuoguAcceptedProblemInput{
			{
				Handle:      "qiaochu",
				ProblemKey:  "P1001",
				ProblemID:   "P1001",
				ProblemName: "A+B Problem",
				ProblemURL:  "https://www.luogu.com.cn/problem/P1001",
				AcceptedAt:  now,
				Source:      model.SyncSourceLuoguPractice,
				SourceURL:   "https://www.luogu.com.cn/user/809639/practice",
				FetchedAt:   now,
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

func TestMergeLuoguProblemFactKeepsFirstObservedTimes(t *testing.T) {
	t.Parallel()

	existingFirst := time.Unix(1_700_000_000, 0).UTC()
	existingLatest := time.Unix(1_700_000_000, 0).UTC()
	eventRawID := int64(12)
	merged := mergeLuoguProblemFact(
		model.ProblemFact{
			ProblemKey:           "P1001",
			ProblemName:          "A+B Problem",
			ProblemURL:           "https://www.luogu.com.cn/problem/P1001",
			FirstACAt:            existingFirst,
			FirstACSource:        model.SyncSourceLuoguPractice,
			FirstACSubmissionRef: "P1001",
			FirstACEventRawID:    &eventRawID,
			LatestACAt:           existingLatest,
		},
		problemFactAggregate{
			ProblemKey:          "P1001",
			ProblemIndex:        "P1001",
			ProblemName:         "A+B Problem Updated",
			ProblemURL:          "https://www.luogu.com.cn/problem/P1001",
			FirstACAt:           time.Unix(1_700_900_000, 0).UTC(),
			FirstACSource:       model.SyncSourceLuoguPractice,
			FirstACSubmissionID: "P1001",
			LatestACAt:          time.Unix(1_700_900_000, 0).UTC(),
		},
	)

	if !merged.FirstACAt.Equal(existingFirst) {
		t.Fatalf("mergeLuoguProblemFact() first_ac_at = %v, want %v", merged.FirstACAt, existingFirst)
	}
	if !merged.LatestACAt.Equal(existingLatest) {
		t.Fatalf("mergeLuoguProblemFact() latest_ac_at = %v, want %v", merged.LatestACAt, existingLatest)
	}
	if merged.FirstACSubmissionID != "P1001" {
		t.Fatalf("mergeLuoguProblemFact() first_ac_submission_id = %q, want %q", merged.FirstACSubmissionID, "P1001")
	}
}
