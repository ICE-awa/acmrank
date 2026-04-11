package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrPlatformSyncDataNotFound = errors.New("platform sync data not found")

type platformSyncRepositoryDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type platformSyncTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type pgxPlatformSyncTx struct {
	tx pgx.Tx
}

func (t pgxPlatformSyncTx) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t pgxPlatformSyncTx) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t pgxPlatformSyncTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t pgxPlatformSyncTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxPlatformSyncTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type PlatformProfileSnapshotInput struct {
	DisplayName string
	Rating      *int
	MaxRating   *int
	ProfileURL  string
	Source      string
	Payload     []byte
	FetchedAt   time.Time
}

type PlatformAcceptedEventInput struct {
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

type ListPlatformSyncFilter struct {
	Limit  int
	Offset int
}

type PlatformSyncRepository struct {
	db      platformSyncRepositoryDB
	beginTx func(context.Context) (platformSyncTx, error)
}

func NewPlatformSyncRepository(db platformSyncRepositoryDB) *PlatformSyncRepository {
	return &PlatformSyncRepository{
		db: db,
		beginTx: func(ctx context.Context) (platformSyncTx, error) {
			tx, err := db.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return nil, err
			}

			return pgxPlatformSyncTx{tx: tx}, nil
		},
	}
}
