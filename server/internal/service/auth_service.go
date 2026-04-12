package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

var (
	ErrValidation                  = errors.New("validation failed")
	ErrUsernameUnavailable         = errors.New("username already exists")
	ErrEmailUnavailable            = errors.New("email already exists")
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrUnauthorized                = errors.New("unauthorized")
	ErrEmailVerificationRequired   = errors.New("email verification required")
	ErrUserDisabled                = errors.New("user disabled")
	ErrInvalidEmailVerificationTok = errors.New("invalid email verification token")
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

const maxPasswordBytes = 72

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func (e ValidationError) Is(target error) bool {
	return target == ErrValidation
}

func (e ValidationError) Unwrap() error {
	return ErrValidation
}

type AuthUserStore interface {
	Create(ctx context.Context, params repository.CreateUserParams) (model.User, error)
	GetForLogin(ctx context.Context, username string) (model.User, string, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
	MarkEmailVerified(ctx context.Context, id int64, verifiedAt time.Time) (model.User, error)
	DeletePendingVerificationUser(ctx context.Context, id int64) error
}

type AuthStateStore interface {
	SaveRefreshSession(ctx context.Context, session model.RefreshSession) error
	GetRefreshSession(ctx context.Context, sessionID string) (model.RefreshSession, error)
	DeleteRefreshSession(ctx context.Context, sessionID string) error
	SaveEmailVerification(ctx context.Context, userID int64, verification model.EmailVerification) error
	GetEmailVerification(ctx context.Context, token string) (int64, error)
	DeleteEmailVerification(ctx context.Context, token string) error
}

type PasswordManager interface {
	Hash(password string) (string, error)
	Compare(hashedPassword string, password string) error
}

type SessionTokenManager interface {
	IssueSession(user model.User) (model.IssuedSession, error)
	ParseAccessToken(token string) (model.AccessTokenSubject, error)
	ParseRefreshToken(token string) (model.RefreshTokenSubject, error)
}

type RegisterInput struct {
	Username string
	Email    string
	RealName string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

type RegisterResult struct {
	User              model.User
	EmailVerification model.EmailVerification
}

type SessionResult struct {
	User    model.User
	Session model.IssuedSession
}

type AuthService struct {
	userStore          AuthUserStore
	stateStore         AuthStateStore
	passwordManager    PasswordManager
	tokenManager       SessionTokenManager
	emailVerifyTTL     time.Duration
	now                func() time.Time
	newVerificationTok func() (string, error)
}

func NewAuthService(
	userStore AuthUserStore,
	stateStore AuthStateStore,
	passwordManager PasswordManager,
	tokenManager SessionTokenManager,
	emailVerifyTTL time.Duration,
) *AuthService {
	return &AuthService{
		userStore:          userStore,
		stateStore:         stateStore,
		passwordManager:    passwordManager,
		tokenManager:       tokenManager,
		emailVerifyTTL:     emailVerifyTTL,
		now:                time.Now,
		newVerificationTok: authsupport.GenerateOpaqueToken,
	}
}

func (s *AuthService) Register(
	ctx context.Context,
	input RegisterInput,
) (RegisterResult, error) {
	normalized, err := normalizeRegisterInput(input)
	if err != nil {
		return RegisterResult{}, err
	}

	passwordHash, err := s.passwordManager.Hash(normalized.Password)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userStore.Create(ctx, repository.CreateUserParams{
		Username:     normalized.Username,
		Email:        normalized.Email,
		RealName:     normalized.RealName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return RegisterResult{}, mapAuthStoreError(err)
	}

	token, err := s.newVerificationTok()
	if err != nil {
		return RegisterResult{}, fmt.Errorf("generate verification token: %w", err)
	}

	verification := model.EmailVerification{
		Token:     token,
		ExpiresAt: s.now().UTC().Add(s.emailVerifyTTL),
	}
	if err := s.stateStore.SaveEmailVerification(ctx, user.ID, verification); err != nil {
		if cleanupErr := s.userStore.DeletePendingVerificationUser(ctx, user.ID); cleanupErr != nil {
			return RegisterResult{}, errors.Join(
				fmt.Errorf("save email verification: %w", err),
				fmt.Errorf("cleanup pending user: %w", cleanupErr),
			)
		}

		return RegisterResult{}, fmt.Errorf("save email verification: %w", err)
	}

	return RegisterResult{
		User:              user,
		EmailVerification: verification,
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) (model.User, error) {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return model.User{}, ValidationError{Message: "verification token is required"}
	}

	userID, err := s.stateStore.GetEmailVerification(ctx, trimmedToken)
	if err != nil {
		if errors.Is(err, repository.ErrEmailVerificationNotFound) {
			return model.User{}, ErrInvalidEmailVerificationTok
		}

		return model.User{}, fmt.Errorf("load email verification: %w", err)
	}

	user, err := s.userStore.MarkEmailVerified(ctx, userID, s.now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, ErrInvalidEmailVerificationTok
		}
		if errors.Is(err, repository.ErrUserDisabled) {
			return model.User{}, ErrUserDisabled
		}

		return model.User{}, fmt.Errorf("mark email verified: %w", err)
	}

	if err := s.stateStore.DeleteEmailVerification(ctx, trimmedToken); err != nil {
		log.Printf("delete email verification token %q: %v", trimmedToken, err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (SessionResult, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return SessionResult{}, ValidationError{Message: "username is required"}
	}

	if input.Password == "" {
		return SessionResult{}, ValidationError{Message: "password is required"}
	}

	user, passwordHash, err := s.userStore.GetForLogin(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return SessionResult{}, ErrInvalidCredentials
		}

		return SessionResult{}, fmt.Errorf("load login user: %w", err)
	}

	if err := s.passwordManager.Compare(passwordHash, input.Password); err != nil {
		return SessionResult{}, ErrInvalidCredentials
	}

	if err := ensureUserCanStartSession(user); err != nil {
		return SessionResult{}, err
	}

	return s.issueSession(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (SessionResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return SessionResult{}, ErrUnauthorized
	}

	subject, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return SessionResult{}, ErrUnauthorized
	}

	storedSession, err := s.stateStore.GetRefreshSession(ctx, subject.SessionID)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshSessionNotFound) {
			return SessionResult{}, ErrUnauthorized
		}

		return SessionResult{}, fmt.Errorf("load refresh session: %w", err)
	}

	if storedSession.UserID != subject.UserID || storedSession.TokenID != subject.TokenID {
		return SessionResult{}, ErrUnauthorized
	}

	user, err := s.userStore.GetByID(ctx, subject.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return SessionResult{}, ErrUnauthorized
		}

		return SessionResult{}, fmt.Errorf("load refresh user: %w", err)
	}

	if err := ensureUserCanStartSession(user); err != nil {
		return SessionResult{}, err
	}

	result, err := s.issueSession(ctx, user)
	if err != nil {
		return SessionResult{}, err
	}

	if err := s.stateStore.DeleteRefreshSession(ctx, subject.SessionID); err != nil {
		log.Printf("delete stale refresh session %q: %v", subject.SessionID, err)
	}

	return result, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}

	subject, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil
	}

	if err := s.stateStore.DeleteRefreshSession(ctx, subject.SessionID); err != nil {
		return fmt.Errorf("delete refresh session: %w", err)
	}

	return nil
}

func (s *AuthService) Authenticate(ctx context.Context, accessToken string) (model.User, error) {
	if strings.TrimSpace(accessToken) == "" {
		return model.User{}, ErrUnauthorized
	}

	subject, err := s.tokenManager.ParseAccessToken(accessToken)
	if err != nil {
		return model.User{}, ErrUnauthorized
	}

	user, err := s.userStore.GetByID(ctx, subject.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, ErrUnauthorized
		}

		return model.User{}, fmt.Errorf("load authenticated user: %w", err)
	}

	if err := ensureUserCanStartSession(user); err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *AuthService) issueSession(ctx context.Context, user model.User) (SessionResult, error) {
	issuedSession, err := s.tokenManager.IssueSession(user)
	if err != nil {
		return SessionResult{}, fmt.Errorf("issue session tokens: %w", err)
	}

	now := s.now().UTC()
	err = s.stateStore.SaveRefreshSession(ctx, model.RefreshSession{
		SessionID: issuedSession.SessionID,
		UserID:    user.ID,
		TokenID:   issuedSession.RefreshTokenID,
		CreatedAt: now,
		ExpiresAt: issuedSession.RefreshTokenExpiresAt,
	})
	if err != nil {
		return SessionResult{}, fmt.Errorf("persist refresh session: %w", err)
	}

	return SessionResult{
		User:    user,
		Session: issuedSession,
	}, nil
}

func normalizeRegisterInput(input RegisterInput) (RegisterInput, error) {
	normalized := RegisterInput{
		Username: strings.TrimSpace(input.Username),
		Email:    strings.TrimSpace(input.Email),
		RealName: strings.TrimSpace(input.RealName),
		Password: input.Password,
	}

	switch {
	case normalized.Username == "":
		return RegisterInput{}, ValidationError{Message: "username is required"}
	case !usernamePattern.MatchString(normalized.Username):
		return RegisterInput{}, ValidationError{Message: "username must be 3-32 characters and contain only letters, numbers, or underscores"}
	case normalized.Email == "":
		return RegisterInput{}, ValidationError{Message: "email is required"}
	case !isValidEmail(normalized.Email):
		return RegisterInput{}, ValidationError{Message: "email must be a valid address"}
	case normalized.RealName == "":
		return RegisterInput{}, ValidationError{Message: "real_name is required"}
	case len([]rune(normalized.RealName)) > 64:
		return RegisterInput{}, ValidationError{Message: "real_name must be 64 characters or fewer"}
	case len(input.Password) < 8:
		return RegisterInput{}, ValidationError{Message: "password must be at least 8 characters"}
	case len(input.Password) > maxPasswordBytes:
		return RegisterInput{}, ValidationError{Message: "password must be 72 bytes or fewer"}
	}

	return normalized, nil
}

func isValidEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	return address.Address == email
}

func ensureUserCanStartSession(user model.User) error {
	switch user.Status {
	case model.UserStatusPendingVerification:
		return ErrEmailVerificationRequired
	case model.UserStatusDisabled:
		return ErrUserDisabled
	default:
		return nil
	}
}

func mapAuthStoreError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUsernameTaken):
		return ErrUsernameUnavailable
	case errors.Is(err, repository.ErrEmailTaken):
		return ErrEmailUnavailable
	default:
		return err
	}
}
