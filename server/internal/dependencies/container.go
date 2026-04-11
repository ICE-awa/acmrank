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

type dependencyProbe func(context.Context) model.DependencyHealth

type Container struct {
	startedAt       time.Time
	connectTimeout  time.Duration
	database        *pgxpool.Pool
	redis           *redis.Client
	nats            *nats.Conn
	postgresProbeFn dependencyProbe
	redisProbeFn    dependencyProbe
	natsProbeFn     dependencyProbe
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	container := &Container{
		startedAt:      time.Now().UTC(),
		connectTimeout: cfg.ConnectTimeout,
	}

	postgresPool, err := newPostgres(ctx, cfg.DatabaseURL, cfg.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("configure postgres: %w", err)
	}
	container.database = postgresPool
	container.postgresProbeFn = func(probeCtx context.Context) model.DependencyHealth {
		return probePostgres(probeCtx, container.database, container.connectTimeout)
	}

	redisClient, err := newRedis(ctx, cfg.RedisAddr, cfg.ConnectTimeout)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("configure redis: %w", err)
	}
	container.redis = redisClient
	container.redisProbeFn = func(probeCtx context.Context) model.DependencyHealth {
		return probeRedis(probeCtx, container.redis, container.connectTimeout)
	}

	natsConn, err := newNATS(cfg.NATSURL, cfg.ConnectTimeout)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("configure nats: %w", err)
	}
	container.nats = natsConn
	container.natsProbeFn = func(probeCtx context.Context) model.DependencyHealth {
		return probeNATS(container.nats, container.connectTimeout)
	}

	return container, nil
}

func (c *Container) StartedAt() time.Time {
	return c.startedAt
}

func (c *Container) Statuses(ctx context.Context) []model.DependencyHealth {
	return []model.DependencyHealth{
		c.runProbe(ctx, "postgres", c.postgresProbeFn),
		c.runProbe(ctx, "redis", c.redisProbeFn),
		c.runProbe(ctx, "nats", c.natsProbeFn),
	}
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
) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func newRedis(
	ctx context.Context,
	addr string,
	timeout time.Duration,
) (*redis.Client, error) {
	if addr == "" {
		return nil, errors.New("redis address is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(connectCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func newNATS(
	natsURL string,
	timeout time.Duration,
) (*nats.Conn, error) {
	if natsURL == "" {
		return nil, errors.New("nats URL is required")
	}

	conn, err := nats.Connect(
		natsURL,
		nats.Timeout(timeout),
		nats.Name("acmrank-bootstrap"),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func probePostgres(
	ctx context.Context,
	pool *pgxpool.Pool,
	timeout time.Duration,
) model.DependencyHealth {
	status := model.DependencyHealth{
		Name:       "postgres",
		Configured: pool != nil,
	}
	if pool == nil {
		status.Message = "not initialized"
		return status
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := pool.Ping(probeCtx); err != nil {
		status.Message = err.Error()
		return status
	}

	status.Reachable = true
	status.Message = "connected"
	return status
}

func probeRedis(
	ctx context.Context,
	client *redis.Client,
	timeout time.Duration,
) model.DependencyHealth {
	status := model.DependencyHealth{
		Name:       "redis",
		Configured: client != nil,
	}
	if client == nil {
		status.Message = "not initialized"
		return status
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(probeCtx).Err(); err != nil {
		status.Message = err.Error()
		return status
	}

	status.Reachable = true
	status.Message = "connected"
	return status
}

func probeNATS(conn *nats.Conn, timeout time.Duration) model.DependencyHealth {
	status := model.DependencyHealth{
		Name:       "nats",
		Configured: conn != nil,
	}
	if conn == nil {
		status.Message = "not initialized"
		return status
	}

	if err := conn.FlushTimeout(timeout); err != nil {
		status.Message = err.Error()
		return status
	}

	status.Reachable = true
	status.Message = "connected"
	return status
}

func (c *Container) runProbe(
	ctx context.Context,
	name string,
	probe dependencyProbe,
) model.DependencyHealth {
	if probe == nil {
		return model.DependencyHealth{
			Name:       name,
			Configured: false,
			Reachable:  false,
			Message:    "probe not configured",
		}
	}

	return probe(ctx)
}
