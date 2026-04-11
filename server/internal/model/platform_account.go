package model

import "time"

type Platform string

const (
	PlatformCodeforces Platform = "codeforces"
	PlatformAtCoder    Platform = "atcoder"
	PlatformLuogu      Platform = "luogu"
)

func (p Platform) IsSupported() bool {
	switch p {
	case PlatformCodeforces, PlatformAtCoder, PlatformLuogu:
		return true
	default:
		return false
	}
}

type PlatformAccountStatus string

const (
	PlatformAccountStatusPendingReview PlatformAccountStatus = "pending_review"
	PlatformAccountStatusVerified      PlatformAccountStatus = "verified"
	PlatformAccountStatusDisabled      PlatformAccountStatus = "disabled"
	PlatformAccountStatusRejected      PlatformAccountStatus = "rejected"
)

func (s PlatformAccountStatus) IsSupported() bool {
	switch s {
	case PlatformAccountStatusPendingReview,
		PlatformAccountStatusVerified,
		PlatformAccountStatusDisabled,
		PlatformAccountStatusRejected:
		return true
	default:
		return false
	}
}

type PlatformAccountOwner struct {
	ID       int64
	Username string
	RealName string
}

type PlatformAccount struct {
	ID            int64
	SiteUserID    int64
	Platform      Platform
	Handle        string
	DisplayHandle string
	Status        PlatformAccountStatus
	VerifiedAt    *time.Time
	LastSyncedAt  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Owner         PlatformAccountOwner
}
