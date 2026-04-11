package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

type stubAuthUserStore struct {
	createFn            func(context.Context, repository.CreateUserParams) (model.User, error)
	getForLoginFn       func(context.Context, string) (model.User, string, error)
	getByIDFn           func(context.Context, int64) (model.User, error)
	markEmailVerifiedFn func(context.Context, int64, time.Time) (model.User, error)
}

func (s stubAuthUserStore) Create(ctx context.Context, params repository.CreateUserParams) (model.User, error) {
	return s.createFn(ctx, params)
}

func (s stubAuthUserStore) GetForLogin(ctx context.Context, username string) (model.User, string, error) {
	return s.getForLoginFn(ctx, username)
}

func (s stubAuthUserStore) GetByID(ctx context.Context, id int64) (model.User, error) {
	return s.getByIDFn(ctx, id)
}

func (s stubAuthUserStore) MarkEmailVerified(ctx context.Context, id int64, verifiedAt time.Time) (model.User, error) {
	return s.markEmailVerifiedFn(ctx, id, verifiedAt)
}

type stubAuthStateStore struct {
	saveRefreshSessionFn    func(context.Context, model.RefreshSession) error
	getRefreshSessionFn     func(context.Context, string) (model.RefreshSession, error)
	deleteRefreshSessionFn  func(context.Context, string) error
	saveEmailVerificationFn func(context.Context, int64, model.EmailVerification) error
	consumeEmailVerifyFn    func(context.Context, string) (int64, error)
}

func (s stubAuthStateStore) SaveRefreshSession(ctx context.Context, session model.RefreshSession) error {
	return s.saveRefreshSessionFn(ctx, session)
}

func (s stubAuthStateStore) GetRefreshSession(ctx context.Context, sessionID string) (model.RefreshSession, error) {
	return s.getRefreshSessionFn(ctx, sessionID)
}

func (s stubAuthStateStore) DeleteRefreshSession(ctx context.Context, sessionID string) error {
	return s.deleteRefreshSessionFn(ctx, sessionID)
}

func (s stubAuthStateStore) SaveEmailVerification(ctx context.Context, userID int64, verification model.EmailVerification) error {
	return s.saveEmailVerificationFn(ctx, userID, verification)
}

func (s stubAuthStateStore) ConsumeEmailVerification(ctx context.Context, token string) (int64, error) {
	return s.consumeEmailVerifyFn(ctx, token)
}

type stubPasswordManager struct {
	hashFn    func(string) (string, error)
	compareFn func(string, string) error
}

func (s stubPasswordManager) Hash(password string) (string, error) {
	return s.hashFn(password)
}

func (s stubPasswordManager) Compare(hashedPassword string, password string) error {
	return s.compareFn(hashedPassword, password)
}

type stubTokenManager struct {
	issueSessionFn     func(model.User) (model.IssuedSession, error)
	parseAccessTokenFn func(string) (model.AccessTokenSubject, error)
	parseRefreshFn     func(string) (model.RefreshTokenSubject, error)
}

func (s stubTokenManager) IssueSession(user model.User) (model.IssuedSession, error) {
	return s.issueSessionFn(user)
}

func (s stubTokenManager) ParseAccessToken(token string) (model.AccessTokenSubject, error) {
	return s.parseAccessTokenFn(token)
}

func (s stubTokenManager) ParseRefreshToken(token string) (model.RefreshTokenSubject, error) {
	return s.parseRefreshFn(token)
}

func TestAuthServiceRegisterCreatesPendingUserAndVerificationToken(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	service := NewAuthService(
		stubAuthUserStore{
			createFn: func(_ context.Context, params repository.CreateUserParams) (model.User, error) {
				if params.Username != "tourist" || params.Email != "tourist@example.com" {
					t.Fatalf("Create() params = %+v", params)
				}

				return model.User{
					ID:       1,
					Username: params.Username,
					Email:    params.Email,
					RealName: params.RealName,
					Status:   model.UserStatusPendingVerification,
				}, nil
			},
		},
		stubAuthStateStore{
			saveRefreshSessionFn:   func(context.Context, model.RefreshSession) error { return nil },
			getRefreshSessionFn:    func(context.Context, string) (model.RefreshSession, error) { return model.RefreshSession{}, nil },
			deleteRefreshSessionFn: func(context.Context, string) error { return nil },
			saveEmailVerificationFn: func(_ context.Context, userID int64, verification model.EmailVerification) error {
				if userID != 1 {
					t.Fatalf("SaveEmailVerification() userID = %d, want %d", userID, 1)
				}

				if verification.Token != "verify-token" {
					t.Fatalf("SaveEmailVerification() token = %q", verification.Token)
				}

				return nil
			},
			consumeEmailVerifyFn: func(context.Context, string) (int64, error) { return 0, nil },
		},
		stubPasswordManager{
			hashFn: func(password string) (string, error) {
				if password != "password123" {
					t.Fatalf("Hash() password = %q", password)
				}
				return "hashed-password", nil
			},
			compareFn: func(string, string) error { return nil },
		},
		stubTokenManager{},
		24*time.Hour,
	)
	service.now = func() time.Time { return now }
	service.newVerificationTok = func() (string, error) { return "verify-token", nil }

	result, err := service.Register(context.Background(), RegisterInput{
		Username: " tourist ",
		Email:    "tourist@example.com",
		RealName: "Tourist",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if result.User.Username != "tourist" {
		t.Fatalf("Register() username = %q, want %q", result.User.Username, "tourist")
	}

	if result.EmailVerification.Token != "verify-token" {
		t.Fatalf("Register() token = %q, want %q", result.EmailVerification.Token, "verify-token")
	}
}

func TestAuthServiceLoginRejectsPendingVerificationUser(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		stubAuthUserStore{
			createFn: func(context.Context, repository.CreateUserParams) (model.User, error) {
				return model.User{}, nil
			},
			getForLoginFn: func(context.Context, string) (model.User, string, error) {
				return model.User{
					ID:       1,
					Username: "tourist",
					Status:   model.UserStatusPendingVerification,
				}, "hashed-password", nil
			},
			getByIDFn:           func(context.Context, int64) (model.User, error) { return model.User{}, nil },
			markEmailVerifiedFn: func(context.Context, int64, time.Time) (model.User, error) { return model.User{}, nil },
		},
		stubAuthStateStore{
			saveRefreshSessionFn:    func(context.Context, model.RefreshSession) error { return nil },
			getRefreshSessionFn:     func(context.Context, string) (model.RefreshSession, error) { return model.RefreshSession{}, nil },
			deleteRefreshSessionFn:  func(context.Context, string) error { return nil },
			saveEmailVerificationFn: func(context.Context, int64, model.EmailVerification) error { return nil },
			consumeEmailVerifyFn:    func(context.Context, string) (int64, error) { return 0, nil },
		},
		stubPasswordManager{
			hashFn: func(string) (string, error) { return "", nil },
			compareFn: func(string, string) error {
				return nil
			},
		},
		stubTokenManager{},
		24*time.Hour,
	)

	_, err := service.Login(context.Background(), LoginInput{
		Username: "tourist",
		Password: "password123",
	})
	if !errors.Is(err, ErrEmailVerificationRequired) {
		t.Fatalf("Login() error = %v, want %v", err, ErrEmailVerificationRequired)
	}
}

func TestAuthServiceRefreshRotatesSession(t *testing.T) {
	t.Parallel()

	savedSessions := make([]model.RefreshSession, 0, 1)
	deletedSessions := make([]string, 0, 1)
	service := NewAuthService(
		stubAuthUserStore{
			createFn: func(context.Context, repository.CreateUserParams) (model.User, error) {
				return model.User{}, nil
			},
			getForLoginFn: func(context.Context, string) (model.User, string, error) {
				return model.User{}, "", nil
			},
			getByIDFn: func(context.Context, int64) (model.User, error) {
				return model.User{
					ID:       42,
					Username: "neal",
					Status:   model.UserStatusActive,
				}, nil
			},
			markEmailVerifiedFn: func(context.Context, int64, time.Time) (model.User, error) {
				return model.User{}, nil
			},
		},
		stubAuthStateStore{
			saveRefreshSessionFn: func(_ context.Context, session model.RefreshSession) error {
				savedSessions = append(savedSessions, session)
				return nil
			},
			getRefreshSessionFn: func(context.Context, string) (model.RefreshSession, error) {
				return model.RefreshSession{
					SessionID: "old-session",
					UserID:    42,
					TokenID:   "old-token",
				}, nil
			},
			deleteRefreshSessionFn: func(_ context.Context, sessionID string) error {
				deletedSessions = append(deletedSessions, sessionID)
				return nil
			},
			saveEmailVerificationFn: func(context.Context, int64, model.EmailVerification) error { return nil },
			consumeEmailVerifyFn:    func(context.Context, string) (int64, error) { return 0, nil },
		},
		stubPasswordManager{
			hashFn:    func(string) (string, error) { return "", nil },
			compareFn: func(string, string) error { return nil },
		},
		stubTokenManager{
			issueSessionFn: func(user model.User) (model.IssuedSession, error) {
				return model.IssuedSession{
					AccessToken:           "new-access",
					RefreshToken:          "new-refresh",
					AccessTokenExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
					RefreshTokenExpiresAt: time.Now().UTC().Add(24 * time.Hour),
					SessionID:             "new-session",
					RefreshTokenID:        "new-token",
				}, nil
			},
			parseAccessTokenFn: func(string) (model.AccessTokenSubject, error) {
				return model.AccessTokenSubject{}, nil
			},
			parseRefreshFn: func(token string) (model.RefreshTokenSubject, error) {
				if token != "refresh-token" {
					t.Fatalf("ParseRefreshToken() token = %q", token)
				}

				return model.RefreshTokenSubject{
					UserID:    42,
					SessionID: "old-session",
					TokenID:   "old-token",
				}, nil
			},
		},
		24*time.Hour,
	)

	result, err := service.Refresh(context.Background(), "refresh-token")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if result.Session.SessionID != "new-session" {
		t.Fatalf("Refresh() session id = %q, want %q", result.Session.SessionID, "new-session")
	}

	if len(savedSessions) != 1 || savedSessions[0].SessionID != "new-session" {
		t.Fatalf("Refresh() saved sessions = %+v", savedSessions)
	}

	if len(deletedSessions) != 1 || deletedSessions[0] != "old-session" {
		t.Fatalf("Refresh() deleted sessions = %v", deletedSessions)
	}
}

func TestAuthServiceVerifyEmailConsumesToken(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_300, 0).UTC()
	service := NewAuthService(
		stubAuthUserStore{
			createFn: func(context.Context, repository.CreateUserParams) (model.User, error) {
				return model.User{}, nil
			},
			getForLoginFn: func(context.Context, string) (model.User, string, error) {
				return model.User{}, "", nil
			},
			getByIDFn: func(context.Context, int64) (model.User, error) {
				return model.User{}, nil
			},
			markEmailVerifiedFn: func(_ context.Context, id int64, verifiedAt time.Time) (model.User, error) {
				if id != 9 || !verifiedAt.Equal(now) {
					t.Fatalf("MarkEmailVerified() id = %d, verifiedAt = %v", id, verifiedAt)
				}

				return model.User{
					ID:              id,
					Username:        "verified-user",
					Status:          model.UserStatusActive,
					EmailVerifiedAt: &verifiedAt,
				}, nil
			},
		},
		stubAuthStateStore{
			saveRefreshSessionFn:   func(context.Context, model.RefreshSession) error { return nil },
			getRefreshSessionFn:    func(context.Context, string) (model.RefreshSession, error) { return model.RefreshSession{}, nil },
			deleteRefreshSessionFn: func(context.Context, string) error { return nil },
			saveEmailVerificationFn: func(context.Context, int64, model.EmailVerification) error {
				return nil
			},
			consumeEmailVerifyFn: func(_ context.Context, token string) (int64, error) {
				if token != "verify-token" {
					t.Fatalf("ConsumeEmailVerification() token = %q", token)
				}
				return 9, nil
			},
		},
		stubPasswordManager{
			hashFn:    func(string) (string, error) { return "", nil },
			compareFn: func(string, string) error { return nil },
		},
		stubTokenManager{},
		24*time.Hour,
	)
	service.now = func() time.Time { return now }

	user, err := service.VerifyEmail(context.Background(), "verify-token")
	if err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}

	if user.Status != model.UserStatusActive {
		t.Fatalf("VerifyEmail() status = %q, want %q", user.Status, model.UserStatusActive)
	}
}

func TestAuthServiceAuthenticateRejectsInvalidAccessToken(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		stubAuthUserStore{
			createFn:            func(context.Context, repository.CreateUserParams) (model.User, error) { return model.User{}, nil },
			getForLoginFn:       func(context.Context, string) (model.User, string, error) { return model.User{}, "", nil },
			getByIDFn:           func(context.Context, int64) (model.User, error) { return model.User{}, nil },
			markEmailVerifiedFn: func(context.Context, int64, time.Time) (model.User, error) { return model.User{}, nil },
		},
		stubAuthStateStore{
			saveRefreshSessionFn:    func(context.Context, model.RefreshSession) error { return nil },
			getRefreshSessionFn:     func(context.Context, string) (model.RefreshSession, error) { return model.RefreshSession{}, nil },
			deleteRefreshSessionFn:  func(context.Context, string) error { return nil },
			saveEmailVerificationFn: func(context.Context, int64, model.EmailVerification) error { return nil },
			consumeEmailVerifyFn:    func(context.Context, string) (int64, error) { return 0, nil },
		},
		stubPasswordManager{
			hashFn:    func(string) (string, error) { return "", nil },
			compareFn: func(string, string) error { return nil },
		},
		stubTokenManager{
			issueSessionFn: func(model.User) (model.IssuedSession, error) { return model.IssuedSession{}, nil },
			parseAccessTokenFn: func(string) (model.AccessTokenSubject, error) {
				return model.AccessTokenSubject{}, errors.New("bad token")
			},
			parseRefreshFn: func(string) (model.RefreshTokenSubject, error) {
				return model.RefreshTokenSubject{}, nil
			},
		},
		24*time.Hour,
	)

	_, err := service.Authenticate(context.Background(), "bad-token")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want %v", err, ErrUnauthorized)
	}
}
