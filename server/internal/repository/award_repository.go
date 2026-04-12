package repository

import (
	"context"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type awardRepositoryDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type awardTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type pgxAwardTx struct {
	tx pgx.Tx
}

func (t pgxAwardTx) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t pgxAwardTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxAwardTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type AwardRecordInput struct {
	ContestName string
	AwardName   string
	RankText    string
	AwardDate   time.Time
	Source      string
	SourceURL   string
	Notes       string
}

type ReplaceAwardRecordsParams struct {
	SiteUserID int64
	Platform   string
	Records    []AwardRecordInput
}

type ListAwardRecordsFilter struct {
	Limit  int
	Offset int
}

type AwardRepository struct {
	db      awardRepositoryDB
	beginTx func(context.Context) (awardTx, error)
}

func NewAwardRepository(db awardRepositoryDB) *AwardRepository {
	return &AwardRepository{
		db: db,
		beginTx: func(ctx context.Context) (awardTx, error) {
			tx, err := db.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return nil, err
			}

			return pgxAwardTx{tx: tx}, nil
		},
	}
}

func (r *AwardRepository) ReplaceAutoSyncByUserID(
	ctx context.Context,
	params ReplaceAwardRecordsParams,
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

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM award_records
WHERE site_user_id = $1
  AND platform = $2
  AND is_manual = false`,
		params.SiteUserID,
		params.Platform,
	); err != nil {
		return err
	}

	for _, record := range params.Records {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO award_records (
  site_user_id, platform, contest_name, award_name, rank_text, award_date,
  source, source_url, is_manual, notes
)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, ''), false, NULLIF($9, ''))
ON CONFLICT (site_user_id, platform, contest_name, award_name, award_date) DO NOTHING`,
			params.SiteUserID,
			params.Platform,
			record.ContestName,
			record.AwardName,
			record.RankText,
			record.AwardDate.UTC(),
			record.Source,
			record.SourceURL,
			record.Notes,
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	committed = true
	return nil
}

func (r *AwardRepository) ListByUserID(
	ctx context.Context,
	siteUserID int64,
	filter ListAwardRecordsFilter,
) ([]model.AwardRecord, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, site_user_id, platform, contest_name, award_name, COALESCE(rank_text, ''),
       award_date, source, COALESCE(source_url, ''), is_manual, COALESCE(notes, ''), created_at
FROM award_records
WHERE site_user_id = $1
ORDER BY award_date DESC, contest_name ASC, id DESC
LIMIT $2 OFFSET $3`,
		siteUserID,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectAwardRecords(rows)
}

type awardScanner interface {
	Scan(dest ...any) error
}

func scanAwardRecord(scanner awardScanner) (model.AwardRecord, error) {
	var record model.AwardRecord
	err := scanner.Scan(
		&record.ID,
		&record.SiteUserID,
		&record.Platform,
		&record.ContestName,
		&record.AwardName,
		&record.RankText,
		&record.AwardDate,
		&record.Source,
		&record.SourceURL,
		&record.IsManual,
		&record.Notes,
		&record.CreatedAt,
	)
	if err != nil {
		return model.AwardRecord{}, err
	}

	return record, nil
}

func collectAwardRecords(rows pgx.Rows) ([]model.AwardRecord, error) {
	records := make([]model.AwardRecord, 0)
	for rows.Next() {
		record, err := scanAwardRecord(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
