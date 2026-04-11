package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrPlatformAccountNotFound     = errors.New("platform account not found")
	ErrPlatformAccountAlreadyBound = errors.New("platform account already bound")
)

type platformAccountRepositoryDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type CreatePlatformAccountParams struct {
	SiteUserID    int64
	Platform      model.Platform
	Handle        string
	DisplayHandle string
}

type ListPlatformAccountsFilter struct {
	Platform model.Platform
	Status   model.PlatformAccountStatus
}

type ReviewPlatformAccountParams struct {
	AccountID      int64
	ReviewerUserID int64
	Status         model.PlatformAccountStatus
	Reason         string
	ReviewedAt     time.Time
}

type PlatformAccountRepository struct {
	db platformAccountRepositoryDB
}

func NewPlatformAccountRepository(db platformAccountRepositoryDB) *PlatformAccountRepository {
	return &PlatformAccountRepository{db: db}
}

func (r *PlatformAccountRepository) Create(
	ctx context.Context,
	params CreatePlatformAccountParams,
) (model.PlatformAccount, error) {
	account, err := scanPlatformAccount(
		r.db.QueryRow(
			ctx,
			`WITH inserted AS (
	INSERT INTO platform_accounts (site_user_id, platform, handle, display_handle)
	VALUES ($1, $2, $3, $4)
	RETURNING id, site_user_id, platform, handle, display_handle, status, verified_at, last_synced_at, created_at, updated_at
)
SELECT inserted.id, inserted.site_user_id, inserted.platform, inserted.handle, inserted.display_handle,
       inserted.status, inserted.verified_at, inserted.last_synced_at, inserted.created_at, inserted.updated_at,
       users.username, users.real_name
FROM inserted
JOIN users ON users.id = inserted.site_user_id`,
			params.SiteUserID,
			params.Platform,
			params.Handle,
			params.DisplayHandle,
		),
	)
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountPersistenceError(err)
	}

	return account, nil
}

func (r *PlatformAccountRepository) ListByUserID(
	ctx context.Context,
	siteUserID int64,
) ([]model.PlatformAccount, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT pa.id, pa.site_user_id, pa.platform, pa.handle, pa.display_handle,
       pa.status, pa.verified_at, pa.last_synced_at, pa.created_at, pa.updated_at,
       users.username, users.real_name
FROM platform_accounts pa
JOIN users ON users.id = pa.site_user_id
WHERE pa.site_user_id = $1
ORDER BY pa.created_at DESC, pa.id DESC`,
		siteUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectPlatformAccounts(rows)
}

func (r *PlatformAccountRepository) List(
	ctx context.Context,
	filter ListPlatformAccountsFilter,
) ([]model.PlatformAccount, error) {
	var builder strings.Builder
	builder.WriteString(`SELECT pa.id, pa.site_user_id, pa.platform, pa.handle, pa.display_handle,
       pa.status, pa.verified_at, pa.last_synced_at, pa.created_at, pa.updated_at,
       users.username, users.real_name
FROM platform_accounts pa
JOIN users ON users.id = pa.site_user_id`)

	args := make([]any, 0, 2)
	conditions := make([]string, 0, 2)
	if filter.Platform != "" {
		args = append(args, filter.Platform)
		conditions = append(conditions, fmt.Sprintf("pa.platform = $%d", len(args)))
	}

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("pa.status = $%d", len(args)))
	}

	if len(conditions) > 0 {
		builder.WriteString("\nWHERE ")
		builder.WriteString(strings.Join(conditions, " AND "))
	}

	builder.WriteString("\nORDER BY pa.created_at DESC, pa.id DESC")

	rows, err := r.db.Query(ctx, builder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectPlatformAccounts(rows)
}

func (r *PlatformAccountRepository) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	account, err := scanPlatformAccount(
		r.db.QueryRow(
			ctx,
			`SELECT pa.id, pa.site_user_id, pa.platform, pa.handle, pa.display_handle,
       pa.status, pa.verified_at, pa.last_synced_at, pa.created_at, pa.updated_at,
       users.username, users.real_name
FROM platform_accounts pa
JOIN users ON users.id = pa.site_user_id
WHERE pa.id = $1`,
			accountID,
		),
	)
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountPersistenceError(err)
	}

	return account, nil
}

func (r *PlatformAccountRepository) DeleteByUserID(
	ctx context.Context,
	accountID int64,
	siteUserID int64,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`DELETE FROM platform_accounts
WHERE id = $1
  AND site_user_id = $2`,
		accountID,
		siteUserID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrPlatformAccountNotFound
	}

	return nil
}

func (r *PlatformAccountRepository) Review(
	ctx context.Context,
	params ReviewPlatformAccountParams,
) (model.PlatformAccount, error) {
	account, err := scanPlatformAccount(
		r.db.QueryRow(
			ctx,
			`WITH current AS (
	SELECT id, status
	FROM platform_accounts
	WHERE id = $1
),
updated AS (
	UPDATE platform_accounts
	SET status = $3,
	    verified_at = CASE
	        WHEN $3 = 'verified' THEN COALESCE(verified_at, $5)
	        ELSE verified_at
	    END,
	    updated_at = $5
	WHERE id = $1
	RETURNING id, site_user_id, platform, handle, display_handle, status, verified_at, last_synced_at, created_at, updated_at
),
audit AS (
	INSERT INTO platform_account_reviews (platform_account_id, reviewer_user_id, from_status, to_status, reason)
	SELECT current.id, $2, current.status, $3, NULLIF($4, '')
	FROM current
)
SELECT updated.id, updated.site_user_id, updated.platform, updated.handle, updated.display_handle,
       updated.status, updated.verified_at, updated.last_synced_at, updated.created_at, updated.updated_at,
       users.username, users.real_name
FROM updated
JOIN users ON users.id = updated.site_user_id`,
			params.AccountID,
			params.ReviewerUserID,
			params.Status,
			params.Reason,
			params.ReviewedAt.UTC(),
		),
	)
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountPersistenceError(err)
	}

	return account, nil
}

type platformAccountScanner interface {
	Scan(dest ...any) error
}

func collectPlatformAccounts(rows pgx.Rows) ([]model.PlatformAccount, error) {
	accounts := make([]model.PlatformAccount, 0)
	for rows.Next() {
		account, err := scanPlatformAccount(rows)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func scanPlatformAccount(scanner platformAccountScanner) (model.PlatformAccount, error) {
	var account model.PlatformAccount
	var verifiedAt *time.Time
	var lastSyncedAt *time.Time

	err := scanner.Scan(
		&account.ID,
		&account.SiteUserID,
		&account.Platform,
		&account.Handle,
		&account.DisplayHandle,
		&account.Status,
		&verifiedAt,
		&lastSyncedAt,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Owner.Username,
		&account.Owner.RealName,
	)
	if err != nil {
		return model.PlatformAccount{}, err
	}

	account.Owner.ID = account.SiteUserID
	account.VerifiedAt = verifiedAt
	account.LastSyncedAt = lastSyncedAt

	return account, nil
}

func mapPlatformAccountPersistenceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPlatformAccountNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "platform_accounts_platform_handle_key":
			return ErrPlatformAccountAlreadyBound
		}
	}

	return err
}
