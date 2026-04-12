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

type stubAwardHTTPService struct {
	enqueueSyncFn func(context.Context, int64) (model.SyncJob, error)
	listAwardsFn  func(context.Context, int64, service.ListAwardRecordsInput) ([]model.AwardRecord, error)
}

func (s stubAwardHTTPService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
) (model.SyncJob, error) {
	return s.enqueueSyncFn(ctx, siteUserID)
}

func (s stubAwardHTTPService) ListAwards(
	ctx context.Context,
	siteUserID int64,
	input service.ListAwardRecordsInput,
) ([]model.AwardRecord, error) {
	return s.listAwardsFn(ctx, siteUserID, input)
}

func TestAwardHandlerSyncReturnsAcceptedJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_701_050_000, 0).UTC()
	siteUserID := int64(7)
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: siteUserID, Username: "tourist"}))
	handler := NewAwardHandler(stubAwardHTTPService{
		enqueueSyncFn: func(_ context.Context, gotUserID int64) (model.SyncJob, error) {
			if gotUserID != siteUserID {
				t.Fatalf("EnqueueSync() siteUserID=%d, want %d", gotUserID, siteUserID)
			}

			return model.SyncJob{
				ID:          13,
				SiteUserID:  &siteUserID,
				Platform:    model.AwardPlatformICPC,
				JobType:     model.SyncJobTypeICPCAward,
				Status:      model.SyncJobStatusQueued,
				ScheduledAt: now,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
		listAwardsFn: func(context.Context, int64, service.ListAwardRecordsInput) ([]model.AwardRecord, error) {
			return nil, nil
		},
	})
	router.POST("/api/v1/users/me/awards/sync", handler.Sync)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/awards/sync", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestAwardHandlerSyncReturnsBadRequestForValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewAwardHandler(stubAwardHTTPService{
		enqueueSyncFn: func(context.Context, int64) (model.SyncJob, error) {
			return model.SyncJob{}, service.ValidationError{Message: "real_name is required before syncing awards"}
		},
		listAwardsFn: func(context.Context, int64, service.ListAwardRecordsInput) ([]model.AwardRecord, error) {
			return nil, nil
		},
	})
	router.POST("/api/v1/users/me/awards/sync", handler.Sync)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/awards/sync", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAwardHandlerListMineAppliesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_701_050_100, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewAwardHandler(stubAwardHTTPService{
		enqueueSyncFn: func(context.Context, int64) (model.SyncJob, error) {
			return model.SyncJob{}, nil
		},
		listAwardsFn: func(_ context.Context, siteUserID int64, input service.ListAwardRecordsInput) ([]model.AwardRecord, error) {
			if siteUserID != 7 || input.Limit != 20 || input.Offset != 5 {
				t.Fatalf("ListAwards() siteUserID=%d input=%+v", siteUserID, input)
			}

			return []model.AwardRecord{
				{
					ID:          1,
					SiteUserID:  7,
					Platform:    model.AwardPlatformICPC,
					ContestName: "ICPC Asia Regional 2025",
					AwardName:   "Gold Medal",
					RankText:    "Rank 3",
					AwardDate:   time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC),
					Source:      model.SyncSourceICPCAwardsFeed,
					SourceURL:   "https://board.example.test/regional-2025",
					CreatedAt:   now,
				},
			}, nil
		},
	})
	router.GET("/api/v1/users/me/awards", handler.ListMine)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/awards?limit=20&offset=5", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	items, ok := response["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", response["items"])
	}
}
