package model

import "time"

type SyncJobType string

const SyncJobTypeCodeforces SyncJobType = "codeforces_sync"

type SyncJobStatus string

const (
	SyncJobStatusQueued    SyncJobStatus = "queued"
	SyncJobStatusRunning   SyncJobStatus = "running"
	SyncJobStatusSucceeded SyncJobStatus = "succeeded"
	SyncJobStatusFailed    SyncJobStatus = "failed"
	SyncJobStatusCancelled SyncJobStatus = "cancelled"
)

type SyncJob struct {
	ID                int64
	SiteUserID        *int64
	PlatformAccountID *int64
	Platform          string
	JobType           SyncJobType
	Status            SyncJobStatus
	ScheduledAt       time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	AttemptCount      int
	ErrorMessage      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
