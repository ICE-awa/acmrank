package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already exists")
	ErrEmailTaken    = errors.New("email already exists")
)

type userRepositoryDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type CreateUserParams struct {
	Username     string
	Email        string
	RealName     string
	PasswordHash string
}

type UserRepository struct {
	db userRepositoryDB
}

func NewUserRepository(db userRepositoryDB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, params CreateUserParams) (model.User, error) {
	user, err := scanUser(
		r.db.QueryRow(
			ctx,
			`INSERT INTO users (username, email, real_name, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, username, email, real_name, status, email_verified_at, created_at, updated_at`,
			params.Username,
			params.Email,
			params.RealName,
			params.PasswordHash,
		),
	)
	if err != nil {
		return model.User{}, mapUserPersistenceError(err)
	}

	return user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (model.User, error) {
	user, err := scanUser(
		r.db.QueryRow(
			ctx,
			`SELECT id, username, email, real_name, status, email_verified_at, created_at, updated_at
FROM users
WHERE username = $1`,
			username,
		),
	)
	if err != nil {
		return model.User{}, mapUserPersistenceError(err)
	}

	return user, nil
}

func (r *UserRepository) GetForLogin(
	ctx context.Context,
	username string,
) (model.User, string, error) {
	var user model.User
	var passwordHash string
	var emailVerifiedAt *time.Time

	err := r.db.QueryRow(
		ctx,
		`SELECT id, username, email, real_name, status, email_verified_at, created_at, updated_at, password_hash
FROM users
WHERE username = $1`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.RealName,
		&user.Status,
		&emailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
	)
	if err != nil {
		return model.User{}, "", mapUserPersistenceError(err)
	}

	user.EmailVerifiedAt = emailVerifiedAt
	return user, passwordHash, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (model.User, error) {
	user, err := scanUser(
		r.db.QueryRow(
			ctx,
			`SELECT id, username, email, real_name, status, email_verified_at, created_at, updated_at
FROM users
WHERE id = $1`,
			id,
		),
	)
	if err != nil {
		return model.User{}, mapUserPersistenceError(err)
	}

	return user, nil
}

func (r *UserRepository) MarkEmailVerified(
	ctx context.Context,
	id int64,
	verifiedAt time.Time,
) (model.User, error) {
	user, err := scanUser(
		r.db.QueryRow(
			ctx,
			`UPDATE users
SET status = 'active',
    email_verified_at = COALESCE(email_verified_at, $2),
    updated_at = $2
WHERE id = $1
  AND status <> 'disabled'
RETURNING id, username, email, real_name, status, email_verified_at, created_at, updated_at`,
			id,
			verifiedAt.UTC(),
		),
	)
	if err != nil {
		return model.User{}, mapUserPersistenceError(err)
	}

	return user, nil
}

func scanUser(row pgx.Row) (model.User, error) {
	var user model.User
	var emailVerifiedAt *time.Time
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.RealName,
		&user.Status,
		&emailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, err
	}

	user.EmailVerifiedAt = emailVerifiedAt
	return user, nil
}

func mapUserPersistenceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_username_key":
			return ErrUsernameTaken
		case "users_email_key":
			return ErrEmailTaken
		}
	}

	return err
}
