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

type stubPlatformAccountDB struct {
	query      string
	args       []any
	queryFn    func(context.Context, string, ...any) (pgx.Rows, error)
	queryRowFn func(context.Context, string, ...any) pgx.Row
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *stubPlatformAccountDB) Query(
	ctx context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	s.query = query
	s.args = args

	if s.queryFn != nil {
		return s.queryFn(ctx, query, args...)
	}

	return &stubPlatformAccountRows{}, nil
}

func (s *stubPlatformAccountDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	s.query = query
	s.args = args

	if s.queryRowFn != nil {
		return s.queryRowFn(ctx, query, args...)
	}

	return stubRow{}
}

func (s *stubPlatformAccountDB) Exec(
	ctx context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	s.query = query
	s.args = args

	if s.execFn != nil {
		return s.execFn(ctx, query, args...)
	}

	return pgconn.NewCommandTag("DELETE 0"), nil
}

type stubPlatformAccountRows struct {
	index   int
	scanFns []func(dest ...any) error
	err     error
}

func (r *stubPlatformAccountRows) Close() {}

func (r *stubPlatformAccountRows) Err() error {
	return r.err
}

func (r *stubPlatformAccountRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}

func (r *stubPlatformAccountRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *stubPlatformAccountRows) Next() bool {
	return r.index < len(r.scanFns)
}

func (r *stubPlatformAccountRows) Scan(dest ...any) error {
	scanFn := r.scanFns[r.index]
	r.index++
	return scanFn(dest...)
}

func (r *stubPlatformAccountRows) Values() ([]any, error) {
	return nil, nil
}

func (r *stubPlatformAccountRows) RawValues() [][]byte {
	return nil
}

func (r *stubPlatformAccountRows) Conn() *pgx.Conn {
	return nil
}

func TestPlatformAccountRepositoryCreateMapsPersistedAccount(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_100_000, 0).UTC()
	db := &stubPlatformAccountDB{
		queryRowFn: func(_ context.Context, query string, args ...any) pgx.Row {
			if !strings.Contains(query, "INSERT INTO platform_accounts") {
				t.Fatalf("unexpected query: %q", query)
			}

			return stubRow{
				scanFn: func(dest ...any) error {
					assignPlatformAccountRow(dest, model.PlatformAccount{
						ID:            9,
						SiteUserID:    7,
						Platform:      model.PlatformAtCoder,
						Handle:        "tourist",
						DisplayHandle: "tourist",
						Status:        model.PlatformAccountStatusPendingReview,
						CreatedAt:     now,
						UpdatedAt:     now,
						Owner: model.PlatformAccountOwner{
							ID:       7,
							Username: "tourist",
							RealName: "Tourist",
						},
					}, nil, nil)
					return nil
				},
			}
		},
	}

	repository := NewPlatformAccountRepository(db)
	account, err := repository.Create(context.Background(), CreatePlatformAccountParams{
		SiteUserID:    7,
		Platform:      model.PlatformAtCoder,
		Handle:        "tourist",
		DisplayHandle: "tourist",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if account.Status != model.PlatformAccountStatusPendingReview {
		t.Fatalf("Create() status = %q, want %q", account.Status, model.PlatformAccountStatusPendingReview)
	}
}

func TestPlatformAccountRepositoryCreateMapsUniqueConstraint(t *testing.T) {
	t.Parallel()

	repository := NewPlatformAccountRepository(&stubPlatformAccountDB{
		queryRowFn: func(context.Context, string, ...any) pgx.Row {
			return stubRow{
				scanFn: func(dest ...any) error {
					return &pgconn.PgError{
						Code:           "23505",
						ConstraintName: "platform_accounts_platform_handle_key",
					}
				},
			}
		},
	})

	_, err := repository.Create(context.Background(), CreatePlatformAccountParams{})
	if !errors.Is(err, ErrPlatformAccountAlreadyBound) {
		t.Fatalf("Create() error = %v, want %v", err, ErrPlatformAccountAlreadyBound)
	}
}

func TestPlatformAccountRepositoryListByUserIDReturnsAccounts(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_100_100, 0).UTC()
	repository := NewPlatformAccountRepository(&stubPlatformAccountDB{
		queryFn: func(_ context.Context, query string, args ...any) (pgx.Rows, error) {
			if !strings.Contains(query, "WHERE pa.site_user_id = $1") {
				t.Fatalf("unexpected query: %q", query)
			}

			return &stubPlatformAccountRows{
				scanFns: []func(dest ...any) error{
					func(dest ...any) error {
						assignPlatformAccountRow(dest, model.PlatformAccount{
							ID:            1,
							SiteUserID:    3,
							Platform:      model.PlatformCodeforces,
							Handle:        "ecnerwala",
							DisplayHandle: "ecnerwala",
							Status:        model.PlatformAccountStatusVerified,
							CreatedAt:     now,
							UpdatedAt:     now,
							Owner: model.PlatformAccountOwner{
								ID:       3,
								Username: "neal",
								RealName: "Neal",
							},
						}, &now, nil)
						return nil
					},
				},
			}, nil
		},
	})

	accounts, err := repository.ListByUserID(context.Background(), 3)
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}

	if len(accounts) != 1 || accounts[0].Owner.Username != "neal" {
		t.Fatalf("ListByUserID() accounts = %+v", accounts)
	}
}

func TestPlatformAccountRepositoryGetByIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	repository := NewPlatformAccountRepository(&stubPlatformAccountDB{
		queryRowFn: func(context.Context, string, ...any) pgx.Row {
			return stubRow{
				scanFn: func(dest ...any) error {
					return pgx.ErrNoRows
				},
			}
		},
	})

	_, err := repository.GetByID(context.Background(), 1)
	if !errors.Is(err, ErrPlatformAccountNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ErrPlatformAccountNotFound)
	}
}

func TestPlatformAccountRepositoryDeleteByUserIDDeletesOwnedAccount(t *testing.T) {
	t.Parallel()

	repository := NewPlatformAccountRepository(&stubPlatformAccountDB{
		execFn: func(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
			if !strings.Contains(query, "DELETE FROM platform_accounts") {
				t.Fatalf("unexpected query: %q", query)
			}

			return pgconn.NewCommandTag("DELETE 1"), nil
		},
	})

	if err := repository.DeleteByUserID(context.Background(), 9, 7); err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}
}

func TestPlatformAccountRepositoryReviewUpdatesStatusAndRecordsAudit(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_100_200, 0).UTC()
	db := &stubPlatformAccountDB{
		queryRowFn: func(_ context.Context, query string, args ...any) pgx.Row {
			if !strings.Contains(query, "INSERT INTO platform_account_reviews") {
				t.Fatalf("Review() query missing audit insert: %q", query)
			}

			return stubRow{
				scanFn: func(dest ...any) error {
					assignPlatformAccountRow(dest, model.PlatformAccount{
						ID:            5,
						SiteUserID:    2,
						Platform:      model.PlatformLuogu,
						Handle:        "P12345",
						DisplayHandle: "P12345",
						Status:        model.PlatformAccountStatusDisabled,
						CreatedAt:     now.Add(-time.Hour),
						UpdatedAt:     now,
						Owner: model.PlatformAccountOwner{
							ID:       2,
							Username: "alice",
							RealName: "Alice",
						},
					}, &now, nil)
					return nil
				},
			}
		},
	}

	repository := NewPlatformAccountRepository(db)
	account, err := repository.Review(context.Background(), ReviewPlatformAccountParams{
		AccountID:      5,
		ReviewerUserID: 1,
		Status:         model.PlatformAccountStatusDisabled,
		Reason:         "ownership mismatch",
		ReviewedAt:     now,
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if account.Status != model.PlatformAccountStatusDisabled {
		t.Fatalf("Review() status = %q, want %q", account.Status, model.PlatformAccountStatusDisabled)
	}
}

func assignPlatformAccountRow(
	dest []any,
	account model.PlatformAccount,
	verifiedAt *time.Time,
	lastSyncedAt *time.Time,
) {
	*(dest[0].(*int64)) = account.ID
	*(dest[1].(*int64)) = account.SiteUserID
	*(dest[2].(*model.Platform)) = account.Platform
	*(dest[3].(*string)) = account.Handle
	*(dest[4].(*string)) = account.DisplayHandle
	*(dest[5].(*model.PlatformAccountStatus)) = account.Status
	*(dest[6].(**time.Time)) = verifiedAt
	*(dest[7].(**time.Time)) = lastSyncedAt
	*(dest[8].(*time.Time)) = account.CreatedAt
	*(dest[9].(*time.Time)) = account.UpdatedAt
	*(dest[10].(*string)) = account.Owner.Username
	*(dest[11].(*string)) = account.Owner.RealName
}
