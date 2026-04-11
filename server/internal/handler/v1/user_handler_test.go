package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

func TestUserHandlerGetMeReturnsAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_000_000, 0).UTC()
	authService := stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{}, nil
		},
		refreshFn: func(context.Context, string) (service.SessionResult, error) { return service.SessionResult{}, nil },
		logoutFn:  func(context.Context, string) error { return nil },
		authenticateFn: func(context.Context, string) (model.User, error) {
			return model.User{
				ID:        7,
				Username:  "ecnerwala",
				Email:     "ecnerwala@example.com",
				RealName:  "Ecnerwala",
				Status:    model.UserStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	router := gin.New()
	middleware := NewAuthMiddleware(authService)
	handler := NewUserHandler()
	router.GET("/api/v1/users/me", middleware.RequireAuthenticated(), handler.GetMe)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  authsupport.AccessTokenCookieName,
		Value: "access-token",
	})
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRejectsUnauthorizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	middleware := NewAuthMiddleware(stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{}, nil
		},
		refreshFn: func(context.Context, string) (service.SessionResult, error) { return service.SessionResult{}, nil },
		logoutFn:  func(context.Context, string) error { return nil },
		authenticateFn: func(context.Context, string) (model.User, error) {
			return model.User{}, service.ErrUnauthorized
		},
	})
	handler := NewUserHandler()
	router.GET("/api/v1/users/me", middleware.RequireAuthenticated(), handler.GetMe)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareRequireAdminRejectsNonAdminUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(authenticatedUserContextKey, model.User{
			ID:       1,
			Username: "tourist",
			Status:   model.UserStatusActive,
		})
		c.Next()
	})

	middleware := NewAuthMiddleware(stubAuthHTTPService{}, "admin")
	router.GET("/api/v1/admin/ping", middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ping", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthMiddlewareRequireAdminAllowsConfiguredAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(authenticatedUserContextKey, model.User{
			ID:       2,
			Username: "Admin",
			Status:   model.UserStatusActive,
		})
		c.Next()
	})

	middleware := NewAuthMiddleware(stubAuthHTTPService{}, "admin")
	router.GET("/api/v1/admin/ping", middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ping", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
