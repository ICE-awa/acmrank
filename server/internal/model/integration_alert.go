package model

import "time"

type IntegrationName string

const (
	IntegrationAtCoderMain       IntegrationName = "atcoder_main"
	IntegrationAtCoderClist      IntegrationName = "atcoder_clist"
	IntegrationAtCoderThirdParty IntegrationName = "atcoder_third_party"
)

type IntegrationAlertStatus string

const (
	IntegrationAlertStatusActive   IntegrationAlertStatus = "active"
	IntegrationAlertStatusResolved IntegrationAlertStatus = "resolved"
)

type IntegrationAlert struct {
	ID               int64
	Integration      IntegrationName
	Status           IntegrationAlertStatus
	Summary          string
	Detail           string
	FirstTriggeredAt time.Time
	LastTriggeredAt  time.Time
	ResolvedAt       *time.Time
	FailureCount     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
