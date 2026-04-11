package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type stubAuthHTTPService struct {
	registerFn     func(context.Context, service.RegisterInput) (service.RegisterResult, error)
	verifyEmailFn  func(context.Context, string) (model.User, error)
	loginFn        func(context.Context, service.LoginInput) (service.SessionResult, error)
	refreshFn      func(context.Context, string) (service.SessionResult, error)
	logoutFn       func(context.Context, string) error
	authenticateFn func(context.Context, string) (model.User, error)
}

func (s stubAuthHTTPService) Register(ctx context.Context, input service.RegisterInput) (service.RegisterResult, error) {
	return s.registerFn(ctx, input)
}

func (s stubAuthHTTPService) VerifyEmail(ctx context.Context, token string) (model.User, error) {
	return s.verifyEmailFn(ctx, token)
}

func (s stubAuthHTTPService) Login(ctx context.Context, input service.LoginInput) (service.SessionResult, error) {
	return s.loginFn(ctx, input)
}

func (s stubAuthHTTPService) Refresh(ctx context.Context, refreshToken string) (service.SessionResult, error) {
	return s.refreshFn(ctx, refreshToken)
}

func (s stubAuthHTTPService) Logout(ctx context.Context, refreshToken string) error {
	return s.logoutFn(ctx, refreshToken)
}

func (s stubAuthHTTPService) Authenticate(ctx context.Context, accessToken string) (model.User, error) {
	return s.authenticateFn(ctx, accessToken)
}

func TestAuthHandlerRegisterReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_000_000, 0).UTC()
	router := gin.New()
	handler := NewAuthHandler(stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{
				User: model.User{
					ID:        1,
					Username:  "tourist",
					Email:     "tourist@example.com",
					RealName:  "Tourist",
					Status:    model.UserStatusPendingVerification,
					CreatedAt: now,
					UpdatedAt: now,
				},
				EmailVerification: model.EmailVerification{
					Token:     "verify-token",
					ExpiresAt: now.Add(24 * time.Hour),
				},
			}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{}, nil
		},
		refreshFn:      func(context.Context, string) (service.SessionResult, error) { return service.SessionResult{}, nil },
		logoutFn:       func(context.Context, string) error { return nil },
		authenticateFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
	}, false)
	router.POST("/api/v1/auth/register", handler.Register)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"username":"tourist","email":"tourist@example.com","real_name":"Tourist","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response["email_verification_token"] != "verify-token" {
		t.Fatalf("email_verification_token = %v", response["email_verification_token"])
	}
}

func TestAuthHandlerLoginSetsSessionCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().UTC()
	router := gin.New()
	handler := NewAuthHandler(stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{
				User: model.User{
					ID:        9,
					Username:  "neal",
					Email:     "neal@example.com",
					RealName:  "Neal",
					Status:    model.UserStatusActive,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Session: model.IssuedSession{
					AccessToken:           "access-token",
					RefreshToken:          "refresh-token",
					AccessTokenExpiresAt:  now.Add(15 * time.Minute),
					RefreshTokenExpiresAt: now.Add(24 * time.Hour),
				},
			}, nil
		},
		refreshFn:      func(context.Context, string) (service.SessionResult, error) { return service.SessionResult{}, nil },
		logoutFn:       func(context.Context, string) error { return nil },
		authenticateFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
	}, false)
	router.POST("/api/v1/auth/login", handler.Login)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"neal","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies len = %d, want 2", len(cookies))
	}
}

func TestAuthHandlerLogoutClearsCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(stubAuthHTTPService{
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
			return model.User{}, nil
		},
	}, false)
	router.POST("/api/v1/auth/logout", handler.Logout)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.MaxAge != -1 {
			t.Fatalf("cookie %q MaxAge = %d, want -1", cookie.Name, cookie.MaxAge)
		}
	}
}

func TestAuthHandlerLogoutClearsCookiesEvenWhenServerSideLogoutFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{}, nil
		},
		refreshFn: func(context.Context, string) (service.SessionResult, error) { return service.SessionResult{}, nil },
		logoutFn: func(context.Context, string) error {
			return errors.New("redis unavailable")
		},
		authenticateFn: func(context.Context, string) (model.User, error) {
			return model.User{}, nil
		},
	}, false)
	router.POST("/api/v1/auth/logout", handler.Logout)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "acmrank_at", Value: "access-token"})
	req.AddCookie(&http.Cookie{Name: "acmrank_rt", Value: "refresh-token"})
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies len = %d, want 2", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.MaxAge != -1 {
			t.Fatalf("cookie %q MaxAge = %d, want -1", cookie.Name, cookie.MaxAge)
		}
	}
}

func TestAuthHandlerMapsUnauthorizedRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(stubAuthHTTPService{
		registerFn: func(context.Context, service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, nil
		},
		verifyEmailFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
		loginFn: func(context.Context, service.LoginInput) (service.SessionResult, error) {
			return service.SessionResult{}, nil
		},
		refreshFn: func(context.Context, string) (service.SessionResult, error) {
			return service.SessionResult{}, service.ErrUnauthorized
		},
		logoutFn:       func(context.Context, string) error { return nil },
		authenticateFn: func(context.Context, string) (model.User, error) { return model.User{}, nil },
	}, false)
	router.POST("/api/v1/auth/refresh", handler.Refresh)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
