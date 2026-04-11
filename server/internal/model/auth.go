package model

import "time"

type RefreshSession struct {
	SessionID string
	UserID    int64
	TokenID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type IssuedSession struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	SessionID             string
	RefreshTokenID        string
}

type EmailVerification struct {
	Token     string
	ExpiresAt time.Time
}

type AccessTokenSubject struct {
	UserID   int64
	Username string
}

type RefreshTokenSubject struct {
	UserID    int64
	SessionID string
	TokenID   string
}
