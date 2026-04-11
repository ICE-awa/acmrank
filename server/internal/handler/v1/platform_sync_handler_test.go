package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/gin-gonic/gin"
)

type stubPlatformSyncHTTPService struct {
	enqueueSyncFn func(context.Context, int64, int64) (model.SyncJob, error)
}

func (s stubPlatformSyncHTTPService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	return s.enqueueSyncFn(ctx, siteUserID, accountID)
}

func TestPlatformSyncHandlerReturnsAcceptedJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Unix(1_700_950_000, 0).UTC()
	router := gin.New()
	router.Use(withAuthenticatedUser(model.User{ID: 7, Username: "tourist"}))
	handler := NewPlatformSyncHandler(stubPlatformSyncHTTPService{
		enqueueSyncFn: func(_ context.Context, siteUserID int64, accountID int64) (model.SyncJob, error) {
			if siteUserID != 7 || accountID != 4 {
				t.Fatalf("EnqueueSync() siteUserID=%d accountID=%d", siteUserID, accountID)
			}

			return model.SyncJob{
				ID:                9,
				PlatformAccountID: &accountID,
				Platform:          "luogu",
				JobType:           model.SyncJobTypeLuogu,
				Status:            model.SyncJobStatusQueued,
				ScheduledAt:       now,
				CreatedAt:         now,
				UpdatedAt:         now,
			}, nil
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
