package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenCookieName  = "acmrank_at"
	RefreshTokenCookieName = "acmrank_rt"
)

var randomBytes = func(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}

	return bytes, nil
}

type TokenManager struct {
	issuer        string
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	now           func() time.Time
}

type tokenClaims struct {
	TokenType string `json:"token_type"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	TokenID   string `json:"token_id,omitempty"`
	jwt.RegisteredClaims
}

func NewTokenManager(
	issuer string,
	accessSecret string,
	refreshSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (*TokenManager, error) {
	switch {
	case issuer == "":
		return nil, errors.New("token issuer is required")
	case accessSecret == "":
		return nil, errors.New("access token secret is required")
	case refreshSecret == "":
		return nil, errors.New("refresh token secret is required")
	case accessTTL <= 0:
		return nil, errors.New("access token ttl must be positive")
	case refreshTTL <= 0:
		return nil, errors.New("refresh token ttl must be positive")
	}

	return &TokenManager{
		issuer:        issuer,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		now:           time.Now,
	}, nil
}

func (m *TokenManager) IssueSession(user model.User) (model.IssuedSession, error) {
	now := m.now().UTC()
	sessionID, err := randomToken()
	if err != nil {
		return model.IssuedSession{}, fmt.Errorf("generate session id: %w", err)
	}

	refreshTokenID, err := randomToken()
	if err != nil {
		return model.IssuedSession{}, fmt.Errorf("generate refresh token id: %w", err)
	}

	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)

	accessToken, err := m.signToken(m.accessSecret, tokenClaims{
		TokenType: "access",
		UserID:    user.ID,
		Username:  user.Username,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
		},
	})
	if err != nil {
		return model.IssuedSession{}, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := m.signToken(m.refreshSecret, tokenClaims{
		TokenType: "refresh",
		UserID:    user.ID,
		SessionID: sessionID,
		TokenID:   refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
		},
	})
	if err != nil {
		return model.IssuedSession{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return model.IssuedSession{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		SessionID:             sessionID,
		RefreshTokenID:        refreshTokenID,
	}, nil
}

func (m *TokenManager) ParseAccessToken(token string) (model.AccessTokenSubject, error) {
	claims, err := m.parseToken(token, m.accessSecret)
	if err != nil {
		return model.AccessTokenSubject{}, err
	}

	if claims.TokenType != "access" {
		return model.AccessTokenSubject{}, errors.New("unexpected access token type")
	}

	return model.AccessTokenSubject{
		UserID:   claims.UserID,
		Username: claims.Username,
	}, nil
}

func (m *TokenManager) ParseRefreshToken(token string) (model.RefreshTokenSubject, error) {
	claims, err := m.parseToken(token, m.refreshSecret)
	if err != nil {
		return model.RefreshTokenSubject{}, err
	}

	if claims.TokenType != "refresh" {
		return model.RefreshTokenSubject{}, errors.New("unexpected refresh token type")
	}

	return model.RefreshTokenSubject{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		TokenID:   claims.TokenID,
	}, nil
}

func (m *TokenManager) signToken(secret []byte, claims tokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (m *TokenManager) parseToken(token string, secret []byte) (tokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(
		token,
		&tokenClaims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
			}

			return secret, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return tokenClaims{}, err
	}

	claims, ok := parsed.Claims.(*tokenClaims)
	if !ok {
		return tokenClaims{}, errors.New("unexpected token claims")
	}

	return *claims, nil
}

func randomToken() (string, error) {
	bytes, err := randomBytes(32)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func GenerateOpaqueToken() (string, error) {
	return randomToken()
}
