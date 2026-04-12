package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	"github.com/ICE-awa/acmrank/server/internal/config"
	"github.com/ICE-awa/acmrank/server/internal/dependencies"
	handlerv1 "github.com/ICE-awa/acmrank/server/internal/handler/v1"
	"github.com/ICE-awa/acmrank/server/internal/integration"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
	"github.com/ICE-awa/acmrank/server/internal/secret"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	config            config.Config
	dependencies      *dependencies.Container
	server            *http.Server
	backgroundWorkers []func(context.Context)
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

	backgroundWorkers := make([]func(context.Context), 0)
	if cfg.Service == appmeta.ServiceAPI {
		apiWorkers, err := registerAPIRoutes(v1, cfg, dependencySet)
		if err != nil {
			_ = dependencySet.Close()
			return nil, err
		}
		backgroundWorkers = append(backgroundWorkers, apiWorkers...)
	}

	server := newHTTPServer(cfg, router)

	return &App{
		config:            cfg,
		dependencies:      dependencySet,
		server:            server,
		backgroundWorkers: backgroundWorkers,
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
) ([]func(context.Context), error) {
	tokenManager, err := authsupport.NewTokenManager(
		"acmrank-api",
		cfg.AccessTokenSecret,
		cfg.RefreshTokenSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("configure token manager: %w", err)
	}

	userRepository := repository.NewUserRepository(dependencySet.Database())
	platformAccountRepository := repository.NewPlatformAccountRepository(dependencySet.Database())
	atCoderSyncRepository := repository.NewAtCoderSyncRepository(dependencySet.Database())
	codeforcesSyncRepository := repository.NewCodeforcesSyncRepository(dependencySet.Database())
	luoguSyncRepository := repository.NewLuoguSyncRepository(dependencySet.Database())
	awardRepository := repository.NewAwardRepository(dependencySet.Database())
	syncJobRepository := repository.NewSyncJobRepository(dependencySet.Database())
	credentialRepository := repository.NewIntegrationCredentialRepository(dependencySet.Database())
	alertRepository := repository.NewIntegrationAlertRepository(dependencySet.Database())
	authStateRepository := repository.NewAuthStateRepository(dependencySet.Redis())

	secretsCipher, err := secret.NewAEAD(cfg.SecretsEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("configure secrets cipher: %w", err)
	}
	if err := seedAtCoderCookieHeader(
		context.Background(),
		credentialRepository,
		secretsCipher,
		cfg.AtCoderCookieHeader,
	); err != nil {
		return nil, fmt.Errorf("seed atcoder cookie header: %w", err)
	}

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
	atCoderClient := integration.NewAtCoderClient(
		cfg.AtCoderBaseURL,
		cfg.AtCoderTimeout,
		atCoderCookieHeaderProvider(credentialRepository, secretsCipher),
	)
	codeforcesClient := integration.NewCodeforcesClient(cfg.CodeforcesAPIBaseURL, cfg.CodeforcesAPITimeout)
	luoguClient := integration.NewLuoguClient(cfg.LuoguBaseURL, cfg.LuoguTimeout)
	icpcAwardClient := integration.NewICPCAwardClient(cfg.ICPCAwardsFeedURL, cfg.ICPCTimeout)
	atCoderService := service.NewAtCoderSyncService(
		platformAccountRepository,
		atCoderSyncRepository,
		syncJobRepository,
		alertRepository,
		atCoderClient,
	)
	codeforcesService := service.NewCodeforcesSyncService(
		platformAccountRepository,
		codeforcesSyncRepository,
		syncJobRepository,
		codeforcesClient,
	)
	luoguService := service.NewLuoguSyncService(
		platformAccountRepository,
		luoguSyncRepository,
		syncJobRepository,
		luoguClient,
	)
	awardService := service.NewICPCAwardService(
		userRepository,
		awardRepository,
		syncJobRepository,
		icpcAwardClient,
	)
	platformSyncService := service.NewPlatformSyncService(
		platformAccountRepository,
		atCoderService,
		codeforcesService,
		luoguService,
	)
	platformSyncHandler := handlerv1.NewPlatformSyncHandler(platformSyncService)
	atCoderHandler := handlerv1.NewAtCoderHandler(atCoderService)
	codeforcesHandler := handlerv1.NewCodeforcesHandler(codeforcesService)
	luoguHandler := handlerv1.NewLuoguHandler(luoguService)
	awardHandler := handlerv1.NewAwardHandler(awardService)
	atCoderSyncRunner := service.NewAtCoderSyncJobRunner(atCoderService, 0)
	codeforcesSyncRunner := service.NewCodeforcesSyncJobRunner(codeforcesService, 0)
	luoguSyncRunner := service.NewLuoguSyncJobRunner(luoguService, 0)
	icpcAwardSyncRunner := service.NewICPCAwardSyncJobRunner(awardService, 0)

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/verify-email", authHandler.VerifyEmail)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)

	usersGroup := v1.Group("/users")
	usersGroup.GET("/me", authMiddleware.RequireAuthenticated(), userHandler.GetMe)
	usersGroup.GET("/me/awards", authMiddleware.RequireAuthenticated(), awardHandler.ListMine)
	usersGroup.POST("/me/awards/sync", authMiddleware.RequireAuthenticated(), awardHandler.Sync)
	usersGroup.GET("/me/atcoder/problem-facts", authMiddleware.RequireAuthenticated(), atCoderHandler.ListProblemFacts)
	usersGroup.GET("/me/atcoder/contest-ac-summaries", authMiddleware.RequireAuthenticated(), atCoderHandler.ListContestSummaries)
	usersGroup.GET("/me/codeforces/problem-facts", authMiddleware.RequireAuthenticated(), codeforcesHandler.ListProblemFacts)
	usersGroup.GET("/me/codeforces/contest-ac-summaries", authMiddleware.RequireAuthenticated(), codeforcesHandler.ListContestSummaries)
	usersGroup.GET("/me/luogu/problem-facts", authMiddleware.RequireAuthenticated(), luoguHandler.ListProblemFacts)

	accountsGroup := v1.Group("/accounts")
	accountsGroup.Use(authMiddleware.RequireAuthenticated())
	accountsGroup.GET("", platformAccountHandler.ListMine)
	accountsGroup.POST("", platformAccountHandler.Create)
	accountsGroup.DELETE("/:id", platformAccountHandler.Delete)
	accountsGroup.POST("/:id/sync", platformSyncHandler.Sync)
	accountsGroup.GET("/:id/atcoder/profile", atCoderHandler.GetLatestProfile)
	accountsGroup.GET("/:id/atcoder/contest-histories", atCoderHandler.ListContestHistories)
	accountsGroup.GET("/:id/codeforces/profile", codeforcesHandler.GetLatestProfile)
	accountsGroup.GET("/:id/codeforces/contest-histories", codeforcesHandler.ListContestHistories)
	accountsGroup.GET("/:id/luogu/profile", luoguHandler.GetLatestProfile)

	adminGroup := v1.Group("/admin")
	adminGroup.Use(authMiddleware.RequireAuthenticated(), authMiddleware.RequireAdmin())
	adminPlatformAccounts := adminGroup.Group("/platform-accounts")
	adminPlatformAccounts.GET("", platformAccountHandler.ListAll)
	adminPlatformAccounts.POST("/:id/verify", platformAccountHandler.Verify)
	adminPlatformAccounts.POST("/:id/disable", platformAccountHandler.Disable)
	adminPlatformAccounts.POST("/:id/reject", platformAccountHandler.Reject)

	return []func(context.Context){
		func(ctx context.Context) {
			atCoderSyncRunner.Start(ctx)
		},
		func(ctx context.Context) {
			codeforcesSyncRunner.Start(ctx)
		},
		func(ctx context.Context) {
			luoguSyncRunner.Start(ctx)
		},
		func(ctx context.Context) {
			icpcAwardSyncRunner.Start(ctx)
		},
	}, nil
}

func seedAtCoderCookieHeader(
	ctx context.Context,
	credentialRepository *repository.IntegrationCredentialRepository,
	cipher *secret.AEAD,
	cookieHeader string,
) error {
	cookieHeader = strings.TrimSpace(cookieHeader)
	if cookieHeader == "" {
		return nil
	}

	ciphertext, err := cipher.Encrypt([]byte(cookieHeader))
	if err != nil {
		return err
	}

	return credentialRepository.Upsert(ctx, repository.UpsertIntegrationCredentialParams{
		Integration:   string(model.IntegrationAtCoderMain),
		CredentialKey: "cookie_header",
		Ciphertext:    ciphertext,
	})
}

func atCoderCookieHeaderProvider(
	credentialRepository *repository.IntegrationCredentialRepository,
	cipher *secret.AEAD,
) integration.AtCoderCookieHeaderProvider {
	return func(ctx context.Context) (string, error) {
		ciphertext, err := credentialRepository.Get(ctx, string(model.IntegrationAtCoderMain), "cookie_header")
		if err != nil {
			return "", err
		}

		plaintext, err := cipher.Decrypt(ciphertext)
		if err != nil {
			return "", err
		}

		return string(plaintext), nil
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	for _, worker := range a.backgroundWorkers {
		worker(workerCtx)
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		cancelWorkers()
		return errors.Join(err, a.dependencies.Close())
	case <-ctx.Done():
		cancelWorkers()
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
