package model

import "time"

type HealthStatus string

const (
	HealthStatusOK       HealthStatus = "ok"
	HealthStatusDegraded HealthStatus = "degraded"
)

type DependencyHealth struct {
	Name       string
	Configured bool
	Reachable  bool
	Message    string
}

type HealthSnapshot struct {
	Service      string
	Version      string
	StartedAt    time.Time
	CheckedAt    time.Time
	Dependencies []DependencyHealth
}

type HealthReport struct {
	Service      string
	Version      string
	Status       HealthStatus
	StartedAt    time.Time
	CheckedAt    time.Time
	Dependencies []DependencyHealth
}
