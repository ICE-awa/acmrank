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

type stubAtCoderHTTPService struct {
	enqueueSyncFn        func(context.Context, int64, int64) (model.SyncJob, error)
	getLatestProfileFn   func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error)
	listContestHistoryFn func(context.Context, int64, int64, service.ListPlatformSyncInput) ([]model.PlatformContestHistory, error)
	listProblemFactsFn   func(context.Context, int64, service.ListPlatformSyncInput) ([]model.ProblemFact, error)
	listContestSummaryFn func(context.Context, int64, service.ListPlatformSyncInput) ([]model.ContestACSummary, error)
}

func (s stubAtCoderHTTPService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	return s.enqueueSyncFn(ctx, siteUserID, accountID)
}

func (s stubAtCoderHTTPService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, siteUserID, accountID)
}

func (s stubAtCoderHTTPService) ListContestHistories(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
	input service.ListPlatformSyncInput,
) ([]model.PlatformContestHistory, error) {
	return s.listContestHistoryFn(ctx, siteUserID, accountID, input)
}

func (s stubAtCoderHTTPService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input service.ListPlatformSyncInput,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, input)
}

func (s stubAtCoderHTTPService) ListContestSummaries(
	ctx context.Context,
	siteUserID int64,
	input service.ListPlatformSyncInput,
) ([]model.ContestACSummary, error) {
	return s.listContestSummaryFn(ctx, siteUserID, input)
}

func TestAtCoderHandlerGetLatestProfileReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_930_000, 0).UTC()
	rating := 3797
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewAtCoderHandler(stubAtCoderHTTPService{
		enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
			return model.SyncJob{}, nil
		},
		getLatestProfileFn: func(_ context.Context, siteUserID int64, accountID int64) (model.PlatformProfileSnapshot, error) {
			if siteUserID != 7 || accountID != 4 {
				t.Fatalf("GetLatestProfile() siteUserID=%d accountID=%d", siteUserID, accountID)
			}

			return model.PlatformProfileSnapshot{
				ID:                1,
				PlatformAccountID: 4,
				Source:            model.SyncSourceAtCoderProfilePage,
				DisplayName:       "tourist",
				Rating:            &rating,
				ProfileURL:        "https://atcoder.jp/users/tourist",
				FetchedAt:         now,
				CreatedAt:         now,
			}, nil
		},
		listContestHistoryFn: func(context.Context, int64, int64, service.ListPlatformSyncInput) ([]model.PlatformContestHistory, error) {
			return nil, nil
		},
		listProblemFactsFn: func(context.Context, int64, service.ListPlatformSyncInput) ([]model.ProblemFact, error) {
			return nil, nil
		},
		listContestSummaryFn: func(context.Context, int64, service.ListPlatformSyncInput) ([]model.ContestACSummary, error) {
			return nil, nil
		},
	})
	router.GET("/api/v1/accounts/:id/atcoder/profile", handler.GetLatestProfile)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/4/atcoder/profile", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	profile, ok := response["profile"].(map[string]any)
	if !ok || profile["display_name"] != "tourist" {
		t.Fatalf("profile = %#v", response["profile"])
	}
}

func TestAtCoderHandlerListProblemFactsAppliesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_930_100, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewAtCoderHandler(stubAtCoderHTTPService{
		enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
			return model.SyncJob{}, nil
		},
		getLatestProfileFn: func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error) {
			return model.PlatformProfileSnapshot{}, nil
		},
		listContestHistoryFn: func(context.Context, int64, int64, service.ListPlatformSyncInput) ([]model.PlatformContestHistory, error) {
			return nil, nil
		},
		listProblemFactsFn: func(_ context.Context, siteUserID int64, input service.ListPlatformSyncInput) ([]model.ProblemFact, error) {
			if siteUserID != 7 || input.Limit != 20 || input.Offset != 5 {
				t.Fatalf("ListProblemFacts() siteUserID=%d input=%+v", siteUserID, input)
			}

			return []model.ProblemFact{
				{
					ID:                   1,
					Platform:             model.PlatformAtCoder,
					ProblemKey:           "agc077_a",
					ContestID:            "agc077",
					ProblemIndexOrTaskID: "agc077_a",
					ProblemName:          "A - Candy",
					ProblemURL:           "https://atcoder.jp/contests/agc077/tasks/agc077_a",
					FirstACAt:            now.Add(-time.Hour),
					FirstACSource:        model.SyncSourceAtCoderSubmission,
					FirstACSubmissionRef: "6001",
					LatestACAt:           now,
					CreatedAt:            now,
					UpdatedAt:            now,
				},
			}, nil
		},
		listContestSummaryFn: func(context.Context, int64, service.ListPlatformSyncInput) ([]model.ContestACSummary, error) {
			return nil, nil
		},
	})
	router.GET("/api/v1/users/me/atcoder/problem-facts", handler.ListProblemFacts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/atcoder/problem-facts?limit=20&offset=5", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}
