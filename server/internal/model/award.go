package model

import "time"

const (
	AwardPlatformICPC        = "icpc"
	SyncSourceICPCAwardsFeed = "icpc_awards_feed"
)

type AwardRecord struct {
	ID          int64
	SiteUserID  int64
	Platform    string
	ContestName string
	AwardName   string
	RankText    string
	AwardDate   time.Time
	Source      string
	SourceURL   string
	IsManual    bool
	Notes       string
	CreatedAt   time.Time
}
