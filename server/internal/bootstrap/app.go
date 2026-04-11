package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	"github.com/ICE-awa/acmrank/server/internal/config"
	"github.com/ICE-awa/acmrank/server/internal/dependencies"
	handlerv1 "github.com/ICE-awa/acmrank/server/internal/handler/v1"
	"github.com/ICE-awa/acmrank/server/internal/integration"
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

	if cfg.Service == appmeta.ServiceAPI {
		if err := registerAPIRoutes(v1, cfg, dependencySet); err != nil {
			_ = dependencySet.Close()
			return nil, err
		}
	}

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

func registerAPIRoutes(
	v1 *gin.RouterGroup,
	cfg config.Config,
	dependencySet *dependencies.Container,
) error {
	tokenManager, err := authsupport.NewTokenManager(
		"acmrank-api",
		cfg.AccessTokenSecret,
		cfg.RefreshTokenSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)
	if err != nil {
		return fmt.Errorf("configure token manager: %w", err)
	}

	userRepository := repository.NewUserRepository(dependencySet.Database())
	platformAccountRepository := repository.NewPlatformAccountRepository(dependencySet.Database())
	codeforcesSyncRepository := repository.NewCodeforcesSyncRepository(dependencySet.Database())
	authStateRepository := repository.NewAuthStateRepository(dependencySet.Redis())
	authService := service.NewAuthService(
		userRepository,
		authStateRepository,
		authsupport.NewPasswordManager(0),
		tokenManager,
		cfg.EmailVerifyTTL,
	)
	authHandler := handlerv1.NewAuthHandler(authService, cfg.CookieSecure)
	authMiddleware := handlerv1.NewAuthMiddleware(authService, cfg.AdminUsernames...)
	userHandler := handlerv1.NewUserHandler()
	platformAccountService := service.NewPlatformAccountService(platformAccountRepository)
	platformAccountHandler := handlerv1.NewPlatformAccountHandler(platformAccountService)
	codeforcesClient := integration.NewCodeforcesClient(cfg.CodeforcesAPIBaseURL, cfg.CodeforcesAPITimeout)
	codeforcesService := service.NewCodeforcesSyncService(
		platformAccountRepository,
		codeforcesSyncRepository,
		codeforcesClient,
	)
	codeforcesHandler := handlerv1.NewCodeforcesHandler(codeforcesService)

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/verify-email", authHandler.VerifyEmail)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)

	usersGroup := v1.Group("/users")
	usersGroup.GET("/me", authMiddleware.RequireAuthenticated(), userHandler.GetMe)
	usersGroup.GET("/me/codeforces/problem-facts", authMiddleware.RequireAuthenticated(), codeforcesHandler.ListProblemFacts)
	usersGroup.GET("/me/codeforces/contest-ac-summaries", authMiddleware.RequireAuthenticated(), codeforcesHandler.ListContestSummaries)

	accountsGroup := v1.Group("/accounts")
	accountsGroup.Use(authMiddleware.RequireAuthenticated())
	accountsGroup.GET("", platformAccountHandler.ListMine)
	accountsGroup.POST("", platformAccountHandler.Create)
	accountsGroup.DELETE("/:id", platformAccountHandler.Delete)
	accountsGroup.POST("/:id/sync", codeforcesHandler.Sync)
	accountsGroup.GET("/:id/codeforces/profile", codeforcesHandler.GetLatestProfile)
	accountsGroup.GET("/:id/codeforces/contest-histories", codeforcesHandler.ListContestHistories)

	adminGroup := v1.Group("/admin")
	adminGroup.Use(authMiddleware.RequireAuthenticated(), authMiddleware.RequireAdmin())
	adminPlatformAccounts := adminGroup.Group("/platform-accounts")
	adminPlatformAccounts.GET("", platformAccountHandler.ListAll)
	adminPlatformAccounts.POST("/:id/verify", platformAccountHandler.Verify)
	adminPlatformAccounts.POST("/:id/disable", platformAccountHandler.Disable)
	adminPlatformAccounts.POST("/:id/reject", platformAccountHandler.Reject)

	return nil
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
