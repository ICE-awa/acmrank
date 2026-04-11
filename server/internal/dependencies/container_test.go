package dependencies

import (
	"context"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

func TestStatusesProbesDependenciesEachTime(t *testing.T) {
	t.Parallel()

	callCount := 0
	container := &Container{
		postgresProbeFn: func(context.Context) model.DependencyHealth {
			callCount++
			return model.DependencyHealth{
				Name:       "postgres",
				Configured: true,
				Reachable:  callCount == 1,
				Message:    "dynamic",
			}
		},
		redisProbeFn: func(context.Context) model.DependencyHealth {
			return model.DependencyHealth{Name: "redis", Configured: true, Reachable: true}
		},
		natsProbeFn: func(context.Context) model.DependencyHealth {
			return model.DependencyHealth{Name: "nats", Configured: true, Reachable: true}
		},
	}

	first := container.Statuses(context.Background())
	second := container.Statuses(context.Background())

	if !first[0].Reachable {
		t.Fatal("Statuses() first probe should reflect the first dynamic value")
	}

	if second[0].Reachable {
		t.Fatal("Statuses() second probe reused stale dependency status")
	}
}

func TestNewPostgresReturnsErrorForMissingConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := newPostgres(context.Background(), "", time.Millisecond); err == nil {
		t.Fatal("newPostgres() expected missing configuration error")
	}
}

func TestNewRedisReturnsErrorForMissingConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := newRedis(context.Background(), "", time.Millisecond); err == nil {
		t.Fatal("newRedis() expected missing configuration error")
	}
}

func TestNewNATSReturnsErrorForMissingConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := newNATS("", time.Millisecond); err == nil {
		t.Fatal("newNATS() expected missing configuration error")
	}
}
