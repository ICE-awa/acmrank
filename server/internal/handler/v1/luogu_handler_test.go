package v1

import (
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

type stubLuoguHTTPService struct {
	getLatestProfileFn func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error)
	listProblemFactsFn func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ProblemFact, error)
}

func (s stubLuoguHTTPService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, siteUserID, accountID)
}

func (s stubLuoguHTTPService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input service.ListCodeforcesSyncInput,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, input)
}

func TestLuoguHandlerGetLatestProfileReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_950_100, 0).UTC()
	rating := 1198
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "qiaochu"}))
	handler := NewLuoguHandler(stubLuoguHTTPService{
		getLatestProfileFn: func(_ context.Context, siteUserID int64, accountID int64) (model.PlatformProfileSnapshot, error) {
			if siteUserID != 7 || accountID != 4 {
				t.Fatalf("GetLatestProfile() siteUserID=%d accountID=%d", siteUserID, accountID)
			}

			return model.PlatformProfileSnapshot{
				ID:                1,
				PlatformAccountID: 4,
				Source:            model.SyncSourceLuoguUserInfo,
				DisplayName:       "qiaochu",
				Rating:            &rating,
				ProfileURL:        "https://www.luogu.com.cn/user/809639",
				FetchedAt:         now,
				CreatedAt:         now,
			}, nil
		},
		listProblemFactsFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ProblemFact, error) {
			return nil, nil
		},
	})
	router.GET("/api/v1/accounts/:id/luogu/profile", handler.GetLatestProfile)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/4/luogu/profile", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	profile, ok := response["profile"].(map[string]any)
	if !ok || profile["display_name"] != "qiaochu" {
		t.Fatalf("profile = %#v", response["profile"])
	}
}

func TestLuoguHandlerListProblemFactsAppliesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_950_200, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "qiaochu"}))
	handler := NewLuoguHandler(stubLuoguHTTPService{
		getLatestProfileFn: func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error) {
			return model.PlatformProfileSnapshot{}, nil
		},
		listProblemFactsFn: func(_ context.Context, siteUserID int64, input service.ListCodeforcesSyncInput) ([]model.ProblemFact, error) {
			if siteUserID != 7 || input.Limit != 20 || input.Offset != 5 {
				t.Fatalf("ListProblemFacts() siteUserID=%d input=%+v", siteUserID, input)
			}

			return []model.ProblemFact{
				{
					ID:                   1,
					Platform:             model.PlatformLuogu,
					ProblemKey:           "P1001",
					ProblemIndexOrTaskID: "P1001",
					ProblemName:          "A+B Problem",
					ProblemURL:           "https://www.luogu.com.cn/problem/P1001",
					FirstACAt:            now.Add(-time.Hour),
					FirstACSource:        model.SyncSourceLuoguPractice,
					FirstACSubmissionRef: "P1001",
					LatestACAt:           now.Add(-time.Hour),
					CreatedAt:            now,
					UpdatedAt:            now,
				},
			}, nil
		},
	})
	router.GET("/api/v1/users/me/luogu/problem-facts", handler.ListProblemFacts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/luogu/problem-facts?limit=20&offset=5", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}
