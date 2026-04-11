package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/redis/go-redis/v9"
)

type stubAuthStateRedis struct {
	setKey   string
	setValue any
	setTTL   time.Duration
	getValue string
	getErr   error
	delKeys  []string
	delErr   error
}

func (s *stubAuthStateRedis) Set(
	_ context.Context,
	key string,
	value any,
	expiration time.Duration,
) *redis.StatusCmd {
	s.setKey = key
	s.setValue = value
	s.setTTL = expiration
	return redis.NewStatusResult("OK", nil)
}

func (s *stubAuthStateRedis) Get(context.Context, string) *redis.StringCmd {
	return redis.NewStringResult(s.getValue, s.getErr)
}

func (s *stubAuthStateRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.delKeys = keys
	return redis.NewIntResult(1, s.delErr)
}

func TestAuthStateRepositoryPersistsRefreshSession(t *testing.T) {
	t.Parallel()

	redisClient := &stubAuthStateRedis{}
	repository := NewAuthStateRepository(redisClient)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	err := repository.SaveRefreshSession(context.Background(), model.RefreshSession{
		SessionID: "session-1",
		UserID:    42,
		TokenID:   "token-1",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("SaveRefreshSession() error = %v", err)
	}

	if redisClient.setKey != "auth:refresh:session-1" {
		t.Fatalf("SaveRefreshSession() key = %q", redisClient.setKey)
	}

	var record refreshSessionRecord
	if err := json.Unmarshal([]byte(redisClient.setValue.(string)), &record); err != nil {
		t.Fatalf("SaveRefreshSession() payload decode error = %v", err)
	}

	if record.UserID != 42 || record.TokenID != "token-1" {
		t.Fatalf("SaveRefreshSession() payload = %+v", record)
	}

	if redisClient.setTTL <= 0 {
		t.Fatalf("SaveRefreshSession() ttl = %v, want positive", redisClient.setTTL)
	}
}

func TestAuthStateRepositoryReturnsMissingRefreshSession(t *testing.T) {
	t.Parallel()

	repository := NewAuthStateRepository(&stubAuthStateRedis{getErr: redis.Nil})

	_, err := repository.GetRefreshSession(context.Background(), "missing")
	if !errors.Is(err, ErrRefreshSessionNotFound) {
		t.Fatalf("GetRefreshSession() error = %v, want %v", err, ErrRefreshSessionNotFound)
	}
}

func TestAuthStateRepositoryGetsEmailVerificationToken(t *testing.T) {
	t.Parallel()

	repository := NewAuthStateRepository(&stubAuthStateRedis{
		getValue: "108",
	})

	userID, err := repository.GetEmailVerification(context.Background(), "verify-token")
	if err != nil {
		t.Fatalf("GetEmailVerification() error = %v", err)
	}

	if userID != 108 {
		t.Fatalf("GetEmailVerification() user id = %d, want %d", userID, 108)
	}
}

func TestAuthStateRepositoryDeleteRefreshSessionUsesRefreshKey(t *testing.T) {
	t.Parallel()

	redisClient := &stubAuthStateRedis{}
	repository := NewAuthStateRepository(redisClient)

	if err := repository.DeleteRefreshSession(context.Background(), "session-7"); err != nil {
		t.Fatalf("DeleteRefreshSession() error = %v", err)
	}

	if len(redisClient.delKeys) != 1 || redisClient.delKeys[0] != "auth:refresh:session-7" {
		t.Fatalf("DeleteRefreshSession() keys = %v", redisClient.delKeys)
	}
}

func TestAuthStateRepositoryDeleteEmailVerificationUsesVerificationKey(t *testing.T) {
	t.Parallel()

	redisClient := &stubAuthStateRedis{}
	repository := NewAuthStateRepository(redisClient)

	if err := repository.DeleteEmailVerification(context.Background(), "verify-token"); err != nil {
		t.Fatalf("DeleteEmailVerification() error = %v", err)
	}

	if len(redisClient.delKeys) != 1 || redisClient.delKeys[0] != "auth:verify-email:verify-token" {
		t.Fatalf("DeleteEmailVerification() keys = %v", redisClient.delKeys)
	}
}

func TestRemainingTTLUsesMinimalDurationForExpiredValue(t *testing.T) {
	t.Parallel()

	ttl := remainingTTL(time.Now().Add(-time.Second))

	if ttl != time.Millisecond {
		t.Fatalf("remainingTTL() = %v, want %v", ttl, time.Millisecond)
	}
}
