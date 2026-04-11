package model

import "time"

type UserStatus string

const (
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusActive              UserStatus = "active"
	UserStatusDisabled            UserStatus = "disabled"
)

type User struct {
	ID              int64
	Username        string
	Email           string
	RealName        string
	Status          UserStatus
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u User) IsActive() bool {
	return u.Status == UserStatusActive
}
