package v1

type AwardRecordResponse struct {
	ID          int64  `json:"id"`
	Platform    string `json:"platform"`
	ContestName string `json:"contest_name"`
	AwardName   string `json:"award_name"`
	RankText    string `json:"rank_text,omitempty"`
	AwardDate   string `json:"award_date"`
	Source      string `json:"source"`
	SourceURL   string `json:"source_url,omitempty"`
	IsManual    bool   `json:"is_manual"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type AwardRecordsResponse struct {
	Items []AwardRecordResponse `json:"items"`
}
