package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/config"
	"github.com/ICE-awa/acmrank/server/internal/dependencies"
	handlerv1 "github.com/ICE-awa/acmrank/server/internal/handler/v1"
	"github.com/ICE-awa/acmrank/server/internal/repository"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	config       config.Config
	dependencies *dependencies.Container
	server       *http.Server
}

func RunService(ctx context.Context, serviceName appmeta.ServiceName) error {
	cfg, err := config.Load(serviceName)
	if err != nil {
		return err
	}

	app, err := NewApp(ctx, cfg)
	if err != nil {
		return err
	}

	log.Printf("starting %s on %s", cfg.Service, cfg.HTTPAddr)
	return app.Run(ctx)
}

func NewApp(ctx context.Context, cfg config.Config) (*App, error) {
	dependencySet, err := dependencies.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	gin.SetMode(cfg.GinMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	healthRepository := repository.NewHealthRepository(
		cfg.Service,
		cfg.Version,
		dependencySet,
	)
	healthService := service.NewHealthService(healthRepository)
	healthHandler := handlerv1.NewHealthHandler(healthService)

	router.GET("/healthz", healthHandler.Get)
	v1 := router.Group("/api/v1")
	v1.GET("/health", healthHandler.Get)

	server := newHTTPServer(cfg, router)

	return &App{
		config:       cfg,
		dependencies: dependencySet,
		server:       server,
	}, nil
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return errors.Join(err, a.dependencies.Close())
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			a.config.ShutdownTimeout,
		)
		defer cancel()

		return errors.Join(
			a.server.Shutdown(shutdownCtx),
			a.dependencies.Close(),
		)
	}
}
