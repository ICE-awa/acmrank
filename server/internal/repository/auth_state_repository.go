package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRefreshSessionNotFound    = errors.New("refresh session not found")
	ErrEmailVerificationNotFound = errors.New("email verification token not found")
)

type authStateRedis interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	GetDel(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type refreshSessionRecord struct {
	UserID    int64     `json:"user_id"`
	TokenID   string    `json:"token_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthStateRepository struct {
	client authStateRedis
}

func NewAuthStateRepository(client authStateRedis) *AuthStateRepository {
	return &AuthStateRepository{client: client}
}

func (r *AuthStateRepository) SaveRefreshSession(
	ctx context.Context,
	session model.RefreshSession,
) error {
	payload, err := json.Marshal(refreshSessionRecord{
		UserID:    session.UserID,
		TokenID:   session.TokenID,
		CreatedAt: session.CreatedAt.UTC(),
		ExpiresAt: session.ExpiresAt.UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal refresh session: %w", err)
	}

	if err := r.client.Set(
		ctx,
		refreshSessionKey(session.SessionID),
		string(payload),
		remainingTTL(session.ExpiresAt),
	).Err(); err != nil {
		return err
	}

	return nil
}

func (r *AuthStateRepository) GetRefreshSession(
	ctx context.Context,
	sessionID string,
) (model.RefreshSession, error) {
	payload, err := r.client.Get(ctx, refreshSessionKey(sessionID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return model.RefreshSession{}, ErrRefreshSessionNotFound
		}

		return model.RefreshSession{}, err
	}

	var record refreshSessionRecord
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return model.RefreshSession{}, fmt.Errorf("decode refresh session: %w", err)
	}

	return model.RefreshSession{
		SessionID: sessionID,
		UserID:    record.UserID,
		TokenID:   record.TokenID,
		CreatedAt: record.CreatedAt,
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (r *AuthStateRepository) DeleteRefreshSession(
	ctx context.Context,
	sessionID string,
) error {
	return r.client.Del(ctx, refreshSessionKey(sessionID)).Err()
}

func (r *AuthStateRepository) SaveEmailVerification(
	ctx context.Context,
	userID int64,
	verification model.EmailVerification,
) error {
	if err := r.client.Set(
		ctx,
		emailVerificationKey(verification.Token),
		strconv.FormatInt(userID, 10),
		remainingTTL(verification.ExpiresAt),
	).Err(); err != nil {
		return err
	}

	return nil
}

func (r *AuthStateRepository) ConsumeEmailVerification(
	ctx context.Context,
	token string,
) (int64, error) {
	value, err := r.client.GetDel(ctx, emailVerificationKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrEmailVerificationNotFound
		}

		return 0, err
	}

	userID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse verification user id: %w", err)
	}

	return userID, nil
}

func refreshSessionKey(sessionID string) string {
	return "auth:refresh:" + sessionID
}

func emailVerificationKey(token string) string {
	return "auth:verify-email:" + token
}

func remainingTTL(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return time.Millisecond
	}

	return ttl
}
