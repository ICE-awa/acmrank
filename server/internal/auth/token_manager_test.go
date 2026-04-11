package auth

import (
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

func TestTokenManagerIssuesAndParsesSessionTokens(t *testing.T) {
	t.Parallel()

	manager, err := NewTokenManager("acmrank-api", "access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	now := time.Now().UTC()
	manager.now = func() time.Time {
		return now
	}

	session, err := manager.IssueSession(model.User{
		ID:       42,
		Username: "tourist",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	accessSubject, err := manager.ParseAccessToken(session.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}

	if accessSubject.UserID != 42 || accessSubject.Username != "tourist" {
		t.Fatalf("ParseAccessToken() subject = %+v", accessSubject)
	}

	refreshSubject, err := manager.ParseRefreshToken(session.RefreshToken)
	if err != nil {
		t.Fatalf("ParseRefreshToken() error = %v", err)
	}

	if refreshSubject.UserID != 42 {
		t.Fatalf("ParseRefreshToken() user id = %d, want %d", refreshSubject.UserID, 42)
	}

	if refreshSubject.SessionID == "" || refreshSubject.TokenID == "" {
		t.Fatalf("ParseRefreshToken() expected session metadata, got %+v", refreshSubject)
	}
}

func TestTokenManagerRejectsWrongTokenType(t *testing.T) {
	t.Parallel()

	manager, err := NewTokenManager("acmrank-api", "access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	session, err := manager.IssueSession(model.User{ID: 7, Username: "ecnerwala"})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	if _, err := manager.ParseAccessToken(session.RefreshToken); err == nil {
		t.Fatal("ParseAccessToken() expected refresh token rejection")
	}

	if _, err := manager.ParseRefreshToken(session.AccessToken); err == nil {
		t.Fatal("ParseRefreshToken() expected access token rejection")
	}
}

func TestNewTokenManagerRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewTokenManager("", "access-secret", "refresh-secret", time.Minute, time.Hour); err == nil {
		t.Fatal("NewTokenManager() expected issuer validation error")
	}
}
