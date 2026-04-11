package model

import "time"

const (
	SyncSourceCodeforcesAPI = "codeforces_api"
	SyncSourceLuoguUserInfo = "luogu_user_info"
	SyncSourceLuoguPractice = "luogu_practice_page"
)

type PlatformProfileSnapshot struct {
	ID                int64
	PlatformAccountID int64
	Source            string
	DisplayName       string
	Rating            *int
	MaxRating         *int
	ProfileURL        string
	RawPayloadRef     string
	FetchedAt         time.Time
	CreatedAt         time.Time
}

type ProblemFact struct {
	ID                   int64
	SiteUserID           int64
	Platform             Platform
	ProblemKey           string
	ContestID            string
	ProblemIndexOrTaskID string
	ProblemName          string
	ProblemURL           string
	FirstACAt            time.Time
	FirstACSource        string
	FirstACSubmissionRef string
	FirstACEventRawID    *int64
	LatestACAt           time.Time
	ClistProblemID       *int64
	ClistContestID       *int64
	ClistRating          *int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ContestACSummary struct {
	ID            int64
	SiteUserID    int64
	Platform      Platform
	ContestID     string
	ContestName   string
	ACProblemKeys []string
	ACCount       int
	FirstACAt     *time.Time
	LastACAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PlatformContestHistory struct {
	ID                int64
	SiteUserID        int64
	PlatformAccountID *int64
	Platform          Platform
	ContestID         string
	ContestName       string
	Rank              *int
	OldRating         *int
	NewRating         *int
	RatingDelta       *int
	ParticipatedAt    time.Time
	Source            string
	SourceURL         string
	RawPayloadRef     string
	FetchedAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
