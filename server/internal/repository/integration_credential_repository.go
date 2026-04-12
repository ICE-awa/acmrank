package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrIntegrationCredentialNotFound = errors.New("integration credential not found")

type integrationCredentialRepositoryDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type UpsertIntegrationCredentialParams struct {
	Integration   string
	CredentialKey string
	Ciphertext    []byte
}

type IntegrationCredentialRepository struct {
	db integrationCredentialRepositoryDB
}

func NewIntegrationCredentialRepository(
	db integrationCredentialRepositoryDB,
) *IntegrationCredentialRepository {
	return &IntegrationCredentialRepository{db: db}
}

func (r *IntegrationCredentialRepository) Upsert(
	ctx context.Context,
	params UpsertIntegrationCredentialParams,
) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO integration_credentials (integration, credential_key, ciphertext)
VALUES ($1, $2, $3)
ON CONFLICT (integration, credential_key) DO UPDATE
SET ciphertext = EXCLUDED.ciphertext,
    updated_at = NOW()`,
		params.Integration,
		params.CredentialKey,
		params.Ciphertext,
	)
	return err
}

func (r *IntegrationCredentialRepository) Get(
	ctx context.Context,
	integration string,
	credentialKey string,
) ([]byte, error) {
	var ciphertext []byte
	err := r.db.QueryRow(
		ctx,
		`SELECT ciphertext
FROM integration_credentials
WHERE integration = $1
  AND credential_key = $2`,
		integration,
		credentialKey,
	).Scan(&ciphertext)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIntegrationCredentialNotFound
		}

		return nil, err
	}

	return append([]byte(nil), ciphertext...), nil
}
