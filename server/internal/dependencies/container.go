package dependencies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/config"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type Container struct {
	startedAt time.Time
	database  *pgxpool.Pool
	redis     *redis.Client
	nats      *nats.Conn
	statuses  []model.DependencyHealth
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	container := &Container{
		startedAt: time.Now().UTC(),
		statuses:  make([]model.DependencyHealth, 0, 3),
	}

	postgresPool, postgresStatus, err := newPostgres(ctx, cfg.DatabaseURL, cfg.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("configure postgres: %w", err)
	}
	container.database = postgresPool
	container.statuses = append(container.statuses, postgresStatus)

	redisClient, redisStatus := newRedis(ctx, cfg.RedisAddr, cfg.ConnectTimeout)
	container.redis = redisClient
	container.statuses = append(container.statuses, redisStatus)

	natsConn, natsStatus := newNATS(cfg.NATSURL, cfg.ConnectTimeout)
	container.nats = natsConn
	container.statuses = append(container.statuses, natsStatus)

	return container, nil
}

func (c *Container) StartedAt() time.Time {
	return c.startedAt
}

func (c *Container) Statuses() []model.DependencyHealth {
	statuses := make([]model.DependencyHealth, len(c.statuses))
	copy(statuses, c.statuses)

	return statuses
}

func (c *Container) Close() error {
	var errs []error

	if c.nats != nil {
		c.nats.Close()
	}

	if c.redis != nil {
		if err := c.redis.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.database != nil {
		c.database.Close()
	}

	return errors.Join(errs...)
}

func newPostgres(
	ctx context.Context,
	databaseURL string,
	timeout time.Duration,
) (*pgxpool.Pool, model.DependencyHealth, error) {
	status := model.DependencyHealth{Name: "postgres"}
	if databaseURL == "" {
		status.Message = "not configured"
		return nil, status, nil
	}

	status.Configured = true

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, status, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		status.Message = err.Error()
		return nil, status, nil
	}

	if err := pool.Ping(connectCtx); err != nil {
		status.Message = err.Error()
		pool.Close()
		return nil, status, nil
	}

	status.Reachable = true
	status.Message = "connected"
	return pool, status, nil
}

func newRedis(
	ctx context.Context,
	addr string,
	timeout time.Duration,
) (*redis.Client, model.DependencyHealth) {
	status := model.DependencyHealth{Name: "redis"}
	if addr == "" {
		status.Message = "not configured"
		return nil, status
	}

	status.Configured = true

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(connectCtx).Err(); err != nil {
		status.Message = err.Error()
		_ = client.Close()
		return nil, status
	}

	status.Reachable = true
	status.Message = "connected"
	return client, status
}

func newNATS(
	natsURL string,
	timeout time.Duration,
) (*nats.Conn, model.DependencyHealth) {
	status := model.DependencyHealth{Name: "nats"}
	if natsURL == "" {
		status.Message = "not configured"
		return nil, status
	}

	status.Configured = true

	conn, err := nats.Connect(
		natsURL,
		nats.Timeout(timeout),
		nats.Name("acmrank-bootstrap"),
	)
	if err != nil {
		status.Message = err.Error()
		return nil, status
	}

	status.Reachable = true
	status.Message = "connected"
	return conn, status
}
