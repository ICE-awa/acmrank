package dependencies

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/config"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type dependencyProbe func(context.Context) model.DependencyHealth

var natsFlushTimeout = func(conn *nats.Conn, timeout time.Duration) error {
	return conn.FlushTimeout(timeout)
}

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

	natsConn, err := newNATS(cfg.Service, cfg.NATSURL, cfg.ConnectTimeout)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("configure nats: %w", err)
	}
	container.nats = natsConn
	container.natsProbeFn = func(probeCtx context.Context) model.DependencyHealth {
		return probeNATS(probeCtx, container.nats, container.connectTimeout)
	}

	return container, nil
}

func (c *Container) StartedAt() time.Time {
	return c.startedAt
}

func (c *Container) Statuses(ctx context.Context) []model.DependencyHealth {
	probes := []struct {
		name  string
		probe dependencyProbe
	}{
		{name: "postgres", probe: c.postgresProbeFn},
		{name: "redis", probe: c.redisProbeFn},
		{name: "nats", probe: c.natsProbeFn},
	}

	statuses := make([]model.DependencyHealth, len(probes))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(probes))

	for index, probe := range probes {
		go func(index int, probeName string, probeFn dependencyProbe) {
			defer waitGroup.Done()
			statuses[index] = c.runProbe(ctx, probeName, probeFn)
		}(index, probe.name, probe.probe)
	}

	waitGroup.Wait()

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
) (*pgxpool.Pool, error) {
	poolConfig, err := newPostgresPoolConfig(databaseURL, timeout)
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
	client := redis.NewClient(newRedisOptions(addr, timeout))

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(connectCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func newNATS(
	service appmeta.ServiceName,
	natsURL string,
	timeout time.Duration,
) (*nats.Conn, error) {
	if natsURL == "" {
		return nil, errors.New("nats URL is required")
	}

	conn, err := nats.Connect(
		natsURL,
		nats.Timeout(timeout),
		nats.Name(natsConnectionName(service)),
	)
	if err != nil {
		return nil, err
	}

	if err := conn.FlushTimeout(timeout); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func natsConnectionName(service appmeta.ServiceName) string {
	return fmt.Sprintf("acmrank-%s", service)
}

func newPostgresPoolConfig(
	databaseURL string,
	timeout time.Duration,
) (*pgxpool.Config, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	poolConfig.ConnConfig.ConnectTimeout = timeout
	return poolConfig, nil
}

func newRedisOptions(addr string, timeout time.Duration) *redis.Options {
	return &redis.Options{
		Addr:        addr,
		DialTimeout: timeout,
	}
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

func probeNATS(
	ctx context.Context,
	conn *nats.Conn,
	timeout time.Duration,
) model.DependencyHealth {
	status := model.DependencyHealth{
		Name:       "nats",
		Configured: conn != nil,
	}
	if conn == nil {
		status.Message = "not initialized"
		return status
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := probeCtx.Err(); err != nil {
		status.Message = err.Error()
		return status
	}

	flushTimeout := timeout
	if deadline, ok := probeCtx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			status.Message = context.DeadlineExceeded.Error()
			return status
		}
		flushTimeout = remaining
	}

	if err := natsFlushTimeout(conn, flushTimeout); err != nil {
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
