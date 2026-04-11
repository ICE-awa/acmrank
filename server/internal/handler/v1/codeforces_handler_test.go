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

type stubCodeforcesHTTPService struct {
	enqueueSyncFn        func(context.Context, int64, int64) (model.SyncJob, error)
	getLatestProfileFn   func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error)
	listContestHistoryFn func(context.Context, int64, int64, service.ListCodeforcesSyncInput) ([]model.PlatformContestHistory, error)
	listProblemFactsFn   func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ProblemFact, error)
	listContestSummaryFn func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ContestACSummary, error)
}

func (s stubCodeforcesHTTPService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	return s.enqueueSyncFn(ctx, siteUserID, accountID)
}

func (s stubCodeforcesHTTPService) GetLatestProfile(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.PlatformProfileSnapshot, error) {
	return s.getLatestProfileFn(ctx, siteUserID, accountID)
}

func (s stubCodeforcesHTTPService) ListContestHistories(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
	input service.ListCodeforcesSyncInput,
) ([]model.PlatformContestHistory, error) {
	return s.listContestHistoryFn(ctx, siteUserID, accountID, input)
}

func (s stubCodeforcesHTTPService) ListProblemFacts(
	ctx context.Context,
	siteUserID int64,
	input service.ListCodeforcesSyncInput,
) ([]model.ProblemFact, error) {
	return s.listProblemFactsFn(ctx, siteUserID, input)
}

func (s stubCodeforcesHTTPService) ListContestSummaries(
	ctx context.Context,
	siteUserID int64,
	input service.ListCodeforcesSyncInput,
) ([]model.ContestACSummary, error) {
	return s.listContestSummaryFn(ctx, siteUserID, input)
}

func TestCodeforcesHandlerSyncReturnsAcceptedJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_500_000, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewCodeforcesHandler(stubCodeforcesHTTPService{
		enqueueSyncFn: func(_ context.Context, siteUserID int64, accountID int64) (model.SyncJob, error) {
			if siteUserID != 7 || accountID != 4 {
				t.Fatalf("EnqueueSync() siteUserID=%d accountID=%d", siteUserID, accountID)
			}

			return model.SyncJob{
				ID:                9,
				PlatformAccountID: &accountID,
				Platform:          "codeforces",
				JobType:           model.SyncJobTypeCodeforces,
				Status:            model.SyncJobStatusQueued,
				ScheduledAt:       now,
				CreatedAt:         now,
				UpdatedAt:         now,
			}, nil
		},
		getLatestProfileFn: func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error) {
			return model.PlatformProfileSnapshot{}, nil
		},
		listContestHistoryFn: func(context.Context, int64, int64, service.ListCodeforcesSyncInput) ([]model.PlatformContestHistory, error) {
			return nil, nil
		},
		listProblemFactsFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ProblemFact, error) {
			return nil, nil
		},
		listContestSummaryFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ContestACSummary, error) {
			return nil, nil
		},
	})
	router.POST("/api/v1/accounts/:id/sync", handler.Sync)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/4/sync", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestCodeforcesHandlerGetLatestProfileReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_500_100, 0).UTC()
	rating := 3500
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewCodeforcesHandler(stubCodeforcesHTTPService{
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
				Source:            model.SyncSourceCodeforcesAPI,
				DisplayName:       "tourist",
				Rating:            &rating,
				ProfileURL:        "https://codeforces.com/profile/tourist",
				FetchedAt:         now,
				CreatedAt:         now,
			}, nil
		},
		listContestHistoryFn: func(context.Context, int64, int64, service.ListCodeforcesSyncInput) ([]model.PlatformContestHistory, error) {
			return nil, nil
		},
		listProblemFactsFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ProblemFact, error) {
			return nil, nil
		},
		listContestSummaryFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ContestACSummary, error) {
			return nil, nil
		},
	})
	router.GET("/api/v1/accounts/:id/codeforces/profile", handler.GetLatestProfile)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/4/codeforces/profile", nil)
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

func TestCodeforcesHandlerListProblemFactsAppliesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_500_200, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewCodeforcesHandler(stubCodeforcesHTTPService{
		enqueueSyncFn: func(context.Context, int64, int64) (model.SyncJob, error) {
			return model.SyncJob{}, nil
		},
		getLatestProfileFn: func(context.Context, int64, int64) (model.PlatformProfileSnapshot, error) {
			return model.PlatformProfileSnapshot{}, nil
		},
		listContestHistoryFn: func(context.Context, int64, int64, service.ListCodeforcesSyncInput) ([]model.PlatformContestHistory, error) {
			return nil, nil
		},
		listProblemFactsFn: func(_ context.Context, siteUserID int64, input service.ListCodeforcesSyncInput) ([]model.ProblemFact, error) {
			if siteUserID != 7 || input.Limit != 20 || input.Offset != 5 {
				t.Fatalf("ListProblemFacts() siteUserID=%d input=%+v", siteUserID, input)
			}

			return []model.ProblemFact{
				{
					ID:                   1,
					Platform:             model.PlatformCodeforces,
					ProblemKey:           "CF-1000A",
					ContestID:            "1000",
					ProblemIndexOrTaskID: "A",
					ProblemName:          "Problem A",
					ProblemURL:           "https://codeforces.com/contest/1000/problem/A",
					FirstACAt:            now.Add(-time.Hour),
					FirstACSource:        model.SyncSourceCodeforcesAPI,
					FirstACSubmissionRef: "1",
					LatestACAt:           now,
					CreatedAt:            now,
					UpdatedAt:            now,
				},
			}, nil
		},
		listContestSummaryFn: func(context.Context, int64, service.ListCodeforcesSyncInput) ([]model.ContestACSummary, error) {
			return nil, nil
		},
	})
	router.GET("/api/v1/users/me/codeforces/problem-facts", handler.ListProblemFacts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/codeforces/problem-facts?limit=20&offset=5", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}
