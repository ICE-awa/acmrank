package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type stubPlatformAccountHTTPService struct {
	listMineFn func(context.Context, int64) ([]model.PlatformAccount, error)
	createFn   func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error)
	deleteFn   func(context.Context, int64, int64) error
	listAllFn  func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error)
	reviewFn   func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error)
}

func (s stubPlatformAccountHTTPService) ListMine(
	ctx context.Context,
	siteUserID int64,
) ([]model.PlatformAccount, error) {
	return s.listMineFn(ctx, siteUserID)
}

func (s stubPlatformAccountHTTPService) Create(
	ctx context.Context,
	siteUserID int64,
	input service.CreatePlatformAccountInput,
) (model.PlatformAccount, error) {
	return s.createFn(ctx, siteUserID, input)
}

func (s stubPlatformAccountHTTPService) Delete(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) error {
	return s.deleteFn(ctx, siteUserID, accountID)
}

func (s stubPlatformAccountHTTPService) ListAll(
	ctx context.Context,
	input service.ListPlatformAccountsInput,
) ([]model.PlatformAccount, error) {
	return s.listAllFn(ctx, input)
}

func (s stubPlatformAccountHTTPService) Review(
	ctx context.Context,
	accountID int64,
	input service.ReviewPlatformAccountInput,
) (model.PlatformAccount, error) {
	return s.reviewFn(ctx, accountID, input)
}

func TestPlatformAccountHandlerListMineReturnsCurrentUsersAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_100_000, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(_ context.Context, siteUserID int64) ([]model.PlatformAccount, error) {
			if siteUserID != 7 {
				t.Fatalf("ListMine() siteUserID = %d, want %d", siteUserID, 7)
			}

			return []model.PlatformAccount{
				{
					ID:            1,
					SiteUserID:    7,
					Platform:      model.PlatformAtCoder,
					Handle:        "tourist",
					DisplayHandle: "tourist",
					Status:        model.PlatformAccountStatusPendingReview,
					CreatedAt:     now,
					UpdatedAt:     now,
				},
			}, nil
		},
		createFn: func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		deleteFn: func(context.Context, int64, int64) error { return nil },
		listAllFn: func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		reviewFn: func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})
	router.GET("/api/v1/accounts", handler.ListMine)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestPlatformAccountHandlerCreateReturnsCreatedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_100_100, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		createFn: func(_ context.Context, siteUserID int64, input service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			if siteUserID != 7 || input.Platform != "atcoder" || input.Handle != "tourist" {
				t.Fatalf("Create() siteUserID = %d input = %+v", siteUserID, input)
			}

			return model.PlatformAccount{
				ID:            3,
				SiteUserID:    7,
				Platform:      model.PlatformAtCoder,
				Handle:        "tourist",
				DisplayHandle: "tourist",
				Status:        model.PlatformAccountStatusPendingReview,
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		},
		deleteFn: func(context.Context, int64, int64) error { return nil },
		listAllFn: func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		reviewFn: func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})
	router.POST("/api/v1/accounts", handler.Create)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString(`{"platform":"atcoder","handle":"tourist"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestPlatformAccountHandlerDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 9, Username: "neal"}))
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		createFn: func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		deleteFn: func(_ context.Context, siteUserID int64, accountID int64) error {
			if siteUserID != 9 || accountID != 4 {
				t.Fatalf("Delete() siteUserID = %d accountID = %d", siteUserID, accountID)
			}
			return nil
		},
		listAllFn: func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		reviewFn: func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})
	router.DELETE("/api/v1/accounts/:id", handler.Delete)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/4", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestPlatformAccountHandlerListAllIncludesOwnerForAdminView(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_100_200, 0).UTC()
	router := gin.New()
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		createFn: func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		deleteFn: func(context.Context, int64, int64) error { return nil },
		listAllFn: func(_ context.Context, input service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			if input.Platform != "atcoder" || input.Status != "pending_review" {
				t.Fatalf("ListAll() input = %+v", input)
			}

			return []model.PlatformAccount{
				{
					ID:            5,
					SiteUserID:    1,
					Platform:      model.PlatformAtCoder,
					Handle:        "tourist",
					DisplayHandle: "tourist",
					Status:        model.PlatformAccountStatusPendingReview,
					CreatedAt:     now,
					UpdatedAt:     now,
					Owner: model.PlatformAccountOwner{
						ID:       1,
						Username: "tourist",
						RealName: "Tourist",
					},
				},
			}, nil
		},
		reviewFn: func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})
	router.GET("/api/v1/admin/platform-accounts", handler.ListAll)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/platform-accounts?platform=atcoder&status=pending_review", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	accounts, ok := response["accounts"].([]any)
	if !ok || len(accounts) != 1 {
		t.Fatalf("accounts = %#v", response["accounts"])
	}
}

func TestPlatformAccountHandlerVerifyUsesCurrentReviewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_100_300, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 11, Username: "admin"}))
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		createFn: func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		deleteFn: func(context.Context, int64, int64) error { return nil },
		listAllFn: func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		reviewFn: func(_ context.Context, accountID int64, input service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			if accountID != 8 || input.Status != "verified" || input.ReviewerUserID != 11 {
				t.Fatalf("Review() accountID = %d input = %+v", accountID, input)
			}

			return model.PlatformAccount{
				ID:            8,
				SiteUserID:    3,
				Platform:      model.PlatformCodeforces,
				Handle:        "ecnerwala",
				DisplayHandle: "ecnerwala",
				Status:        model.PlatformAccountStatusVerified,
				CreatedAt:     now,
				UpdatedAt:     now,
				Owner: model.PlatformAccountOwner{
					ID:       3,
					Username: "neal",
					RealName: "Neal",
				},
			}, nil
		},
	})
	router.POST("/api/v1/admin/platform-accounts/:id/verify", handler.Verify)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/platform-accounts/8/verify", bytes.NewBufferString(`{"reason":"checked"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestPlatformAccountHandlerMapsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewPlatformAccountHandler(stubPlatformAccountHTTPService{
		listMineFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		createFn: func(context.Context, int64, service.CreatePlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		deleteFn: func(context.Context, int64, int64) error {
			return service.ErrPlatformAccountNotFound
		},
		listAllFn: func(context.Context, service.ListPlatformAccountsInput) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		reviewFn: func(context.Context, int64, service.ReviewPlatformAccountInput) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})
	router.DELETE("/api/v1/accounts/:id", handler.Delete)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/99", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func withAuthenticatedUser(user model.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(authenticatedUserContextKey, user)
		c.Next()
	}
}
