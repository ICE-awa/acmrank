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

type stubUserDB struct {
	query      string
	args       []any
	row        pgx.Row
	queryRowFn func(context.Context, string, ...any) pgx.Row
	execFn     func(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *stubUserDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	s.query = query
	s.args = args

	if s.queryRowFn != nil {
		return s.queryRowFn(ctx, query, args...)
	}

	return s.row
}

func (s *stubUserDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	s.query = query
	s.args = args

	if s.execFn != nil {
		return s.execFn(ctx, query, args...)
	}

	return pgconn.NewCommandTag("DELETE 0"), nil
}

type stubRow struct {
	scanFn func(dest ...any) error
}

func (r stubRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

func TestUserRepositoryCreateMapsPersistedUser(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	db := &stubUserDB{
		row: stubRow{
			scanFn: func(dest ...any) error {
				assignUserRow(dest, model.User{
					ID:        7,
					Username:  "tourist",
					Email:     "tourist@example.com",
					RealName:  "Gennady",
					Status:    model.UserStatusPendingVerification,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
				return nil
			},
		},
	}

	repository := NewUserRepository(db)
	user, err := repository.Create(context.Background(), CreateUserParams{
		Username:     "tourist",
		Email:        "tourist@example.com",
		RealName:     "Gennady",
		PasswordHash: "hashed",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.Username != "tourist" || user.Email != "tourist@example.com" {
		t.Fatalf("Create() user = %+v", user)
	}

	if !strings.Contains(db.query, "INSERT INTO users") {
		t.Fatalf("Create() query = %q", db.query)
	}
}

func TestUserRepositoryMapsUniqueConstraintErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name: "username",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "users_username_key",
			},
			wantErr: ErrUsernameTaken,
		},
		{
			name: "email",
			err: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "users_email_key",
			},
			wantErr: ErrEmailTaken,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := NewUserRepository(&stubUserDB{
				row: stubRow{
					scanFn: func(dest ...any) error {
						return testCase.err
					},
				},
			})

			_, err := repository.Create(context.Background(), CreateUserParams{})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestUserRepositoryGetByUsernameReturnsNotFound(t *testing.T) {
	t.Parallel()

	repository := NewUserRepository(&stubUserDB{
		row: stubRow{
			scanFn: func(dest ...any) error {
				return pgx.ErrNoRows
			},
		},
	})

	_, err := repository.GetByUsername(context.Background(), "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetByUsername() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestUserRepositoryGetForLoginReturnsPasswordHash(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_200, 0).UTC()
	repository := NewUserRepository(&stubUserDB{
		row: stubRow{
			scanFn: func(dest ...any) error {
				assignUserRow(dest[:8], model.User{
					ID:        2,
					Username:  "benq",
					Email:     "benq@example.com",
					RealName:  "Ben",
					Status:    model.UserStatusActive,
					CreatedAt: now.Add(-time.Hour),
					UpdatedAt: now,
				}, nil)
				*(dest[8].(*string)) = "hashed-password"
				return nil
			},
		},
	})

	user, passwordHash, err := repository.GetForLogin(context.Background(), "benq")
	if err != nil {
		t.Fatalf("GetForLogin() error = %v", err)
	}

	if user.Username != "benq" || passwordHash != "hashed-password" {
		t.Fatalf("GetForLogin() user = %+v, password hash = %q", user, passwordHash)
	}
}

func TestUserRepositoryMarkEmailVerifiedActivatesUser(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_100, 0).UTC()
	db := &stubUserDB{
		row: stubRow{
			scanFn: func(dest ...any) error {
				assignUserRow(dest, model.User{
					ID:              1,
					Username:        "neal",
					Email:           "neal@example.com",
					RealName:        "Neal",
					Status:          model.UserStatusActive,
					EmailVerifiedAt: &now,
					CreatedAt:       now.Add(-time.Hour),
					UpdatedAt:       now,
				}, &now)
				return nil
			},
		},
	}

	repository := NewUserRepository(db)
	user, err := repository.MarkEmailVerified(context.Background(), 1, now)
	if err != nil {
		t.Fatalf("MarkEmailVerified() error = %v", err)
	}

	if user.Status != model.UserStatusActive {
		t.Fatalf("MarkEmailVerified() status = %q, want %q", user.Status, model.UserStatusActive)
	}

	if user.EmailVerifiedAt == nil || !user.EmailVerifiedAt.Equal(now) {
		t.Fatalf("MarkEmailVerified() email verified at = %v, want %v", user.EmailVerifiedAt, now)
	}
}

func TestUserRepositoryMarkEmailVerifiedReturnsDisabledError(t *testing.T) {
	t.Parallel()

	repository := NewUserRepository(&stubUserDB{
		queryRowFn: func(_ context.Context, query string, args ...any) pgx.Row {
			switch {
			case strings.Contains(query, "UPDATE users"):
				return stubRow{
					scanFn: func(dest ...any) error {
						return pgx.ErrNoRows
					},
				}
			case strings.Contains(query, "SELECT status"):
				return stubRow{
					scanFn: func(dest ...any) error {
						*(dest[0].(*model.UserStatus)) = model.UserStatusDisabled
						return nil
					},
				}
			default:
				t.Fatalf("unexpected query: %q", query)
				return stubRow{}
			}
		},
	})

	_, err := repository.MarkEmailVerified(context.Background(), 1, time.Now().UTC())
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("MarkEmailVerified() error = %v, want %v", err, ErrUserDisabled)
	}
}

func TestUserRepositoryDeletePendingVerificationUserDeletesPendingUser(t *testing.T) {
	t.Parallel()

	repository := NewUserRepository(&stubUserDB{
		execFn: func(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
			if !strings.Contains(query, "DELETE FROM users") {
				t.Fatalf("unexpected query: %q", query)
			}

			return pgconn.NewCommandTag("DELETE 1"), nil
		},
	})

	if err := repository.DeletePendingVerificationUser(context.Background(), 7); err != nil {
		t.Fatalf("DeletePendingVerificationUser() error = %v", err)
	}
}

func assignUserRow(dest []any, user model.User, emailVerifiedAt *time.Time) {
	*(dest[0].(*int64)) = user.ID
	*(dest[1].(*string)) = user.Username
	*(dest[2].(*string)) = user.Email
	*(dest[3].(*string)) = user.RealName
	*(dest[4].(*model.UserStatus)) = user.Status
	*(dest[5].(**time.Time)) = emailVerifiedAt
	*(dest[6].(*time.Time)) = user.CreatedAt
	*(dest[7].(*time.Time)) = user.UpdatedAt
}
