package v1

type SyncJobResponse struct {
	ID                int64   `json:"id"`
	PlatformAccountID *int64  `json:"platform_account_id,omitempty"`
	Platform          string  `json:"platform,omitempty"`
	JobType           string  `json:"job_type"`
	Status            string  `json:"status"`
	ScheduledAt       string  `json:"scheduled_at"`
	StartedAt         *string `json:"started_at,omitempty"`
	FinishedAt        *string `json:"finished_at,omitempty"`
	AttemptCount      int     `json:"attempt_count"`
	ErrorMessage      string  `json:"error_message,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type PlatformProfileSnapshotResponse struct {
	Source      string `json:"source"`
	DisplayName string `json:"display_name"`
	Rating      *int   `json:"rating,omitempty"`
	MaxRating   *int   `json:"max_rating,omitempty"`
	ProfileURL  string `json:"profile_url"`
	FetchedAt   string `json:"fetched_at"`
	CreatedAt   string `json:"created_at"`
}

type PlatformContestHistoryResponse struct {
	ID             int64  `json:"id"`
	ContestID      string `json:"contest_id"`
	ContestName    string `json:"contest_name"`
	Rank           *int   `json:"rank,omitempty"`
	OldRating      *int   `json:"old_rating,omitempty"`
	NewRating      *int   `json:"new_rating,omitempty"`
	RatingDelta    *int   `json:"rating_delta,omitempty"`
	ParticipatedAt string `json:"participated_at"`
	Source         string `json:"source"`
	SourceURL      string `json:"source_url,omitempty"`
	FetchedAt      string `json:"fetched_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ProblemFactResponse struct {
	ID                   int64  `json:"id"`
	Platform             string `json:"platform"`
	ProblemKey           string `json:"problem_key"`
	ContestID            string `json:"contest_id,omitempty"`
	ProblemIndexOrTaskID string `json:"problem_index_or_task_id,omitempty"`
	ProblemName          string `json:"problem_name,omitempty"`
	ProblemURL           string `json:"problem_url,omitempty"`
	FirstACAt            string `json:"first_ac_at"`
	FirstACSource        string `json:"first_ac_source"`
	FirstACSubmissionRef string `json:"first_ac_submission_ref,omitempty"`
	LatestACAt           string `json:"latest_ac_at"`
	ClistRating          *int   `json:"clist_rating,omitempty"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type ContestACSummaryResponse struct {
	ID            int64    `json:"id"`
	Platform      string   `json:"platform"`
	ContestID     string   `json:"contest_id"`
	ContestName   string   `json:"contest_name,omitempty"`
	ACProblemKeys []string `json:"ac_problem_keys"`
	ACCount       int      `json:"ac_count"`
	FirstACAt     *string  `json:"first_ac_at,omitempty"`
	LastACAt      *string  `json:"last_ac_at,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type PlatformContestHistoriesResponse struct {
	Items []PlatformContestHistoryResponse `json:"items"`
}

type ProblemFactsResponse struct {
	Items []ProblemFactResponse `json:"items"`
}

type ContestACSummariesResponse struct {
	Items []ContestACSummaryResponse `json:"items"`
}
