package dependencies

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
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

func TestStatusesRunsProbesConcurrently(t *testing.T) {
	t.Parallel()

	started := make(chan string, 3)
	release := make(chan struct{})
	probe := func(name string) dependencyProbe {
		return func(context.Context) model.DependencyHealth {
			started <- name
			<-release
			return model.DependencyHealth{Name: name, Configured: true, Reachable: true}
		}
	}

	container := &Container{
		postgresProbeFn: probe("postgres"),
		redisProbeFn:    probe("redis"),
		natsProbeFn:     probe("nats"),
	}

	done := make(chan []model.DependencyHealth, 1)
	go func() {
		done <- container.Statuses(context.Background())
	}()

	gotStarted := make([]string, 0, 3)
	timeout := time.After(100 * time.Millisecond)
	for len(gotStarted) < 3 {
		select {
		case name := <-started:
			gotStarted = append(gotStarted, name)
		case <-timeout:
			t.Fatalf("Statuses() did not start all probes concurrently, started %v", gotStarted)
		}
	}

	close(release)

	statuses := <-done
	if len(statuses) != 3 {
		t.Fatalf("Statuses() len = %d, want 3", len(statuses))
	}

	slices.Sort(gotStarted)
	if !slices.Equal(gotStarted, []string{"nats", "postgres", "redis"}) {
		t.Fatalf("Statuses() started probes = %v", gotStarted)
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

	if _, err := newNATS(appmeta.ServiceAPI, "", time.Millisecond); err == nil {
		t.Fatal("newNATS() expected missing configuration error")
	}
}

func TestNewPostgresPoolConfigSetsConnectTimeout(t *testing.T) {
	t.Parallel()

	cfg, err := newPostgresPoolConfig(
		"postgres://acmrank:acmrank_dev@127.0.0.1:5432/acmrank?sslmode=disable",
		7*time.Second,
	)
	if err != nil {
		t.Fatalf("newPostgresPoolConfig() error = %v", err)
	}

	if cfg.ConnConfig.ConnectTimeout != 7*time.Second {
		t.Fatalf(
			"newPostgresPoolConfig() ConnectTimeout = %v, want %v",
			cfg.ConnConfig.ConnectTimeout,
			7*time.Second,
		)
	}
}

func TestNewRedisOptionsSetsDialTimeout(t *testing.T) {
	t.Parallel()

	options := newRedisOptions("127.0.0.1:6379", 9*time.Second)

	if options.DialTimeout != 9*time.Second {
		t.Fatalf("newRedisOptions() DialTimeout = %v, want %v", options.DialTimeout, 9*time.Second)
	}
}

func TestNATSConnectionNameIncludesService(t *testing.T) {
	t.Parallel()

	if got := natsConnectionName(appmeta.ServiceScheduler); got != "acmrank-scheduler" {
		t.Fatalf("natsConnectionName() = %q, want %q", got, "acmrank-scheduler")
	}
}
