package v1

import (
	"context"
	"net/http"
	"time"

	dtov1 "github.com/ICE-awa/acmrank/server/internal/dto/v1"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AtCoderSyncService interface {
	EnqueueSync(ctx context.Context, siteUserID int64, accountID int64) (model.SyncJob, error)
	GetLatestProfile(ctx context.Context, siteUserID int64, accountID int64) (model.PlatformProfileSnapshot, error)
	ListContestHistories(ctx context.Context, siteUserID int64, accountID int64, input service.ListPlatformSyncInput) ([]model.PlatformContestHistory, error)
	ListProblemFacts(ctx context.Context, siteUserID int64, input service.ListPlatformSyncInput) ([]model.ProblemFact, error)
	ListContestSummaries(ctx context.Context, siteUserID int64, input service.ListPlatformSyncInput) ([]model.ContestACSummary, error)
}

type AtCoderHandler struct {
	service AtCoderSyncService
}

func NewAtCoderHandler(service AtCoderSyncService) *AtCoderHandler {
	return &AtCoderHandler{service: service}
}

func (h *AtCoderHandler) Sync(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	accountID, err := parseInt64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	job, err := h.service.EnqueueSync(c.Request.Context(), user.ID, accountID)
	if err != nil {
		writePlatformSyncError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job": toSyncJobResponse(job),
	})
}

func (h *AtCoderHandler) GetLatestProfile(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	accountID, err := parseInt64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	snapshot, err := h.service.GetLatestProfile(c.Request.Context(), user.ID, accountID)
	if err != nil {
		writePlatformSyncError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profile": dtov1.PlatformProfileSnapshotResponse{
			Source:      snapshot.Source,
			DisplayName: snapshot.DisplayName,
			Rating:      snapshot.Rating,
			MaxRating:   snapshot.MaxRating,
			ProfileURL:  snapshot.ProfileURL,
			FetchedAt:   snapshot.FetchedAt.UTC().Format(time.RFC3339),
			CreatedAt:   snapshot.CreatedAt.UTC().Format(time.RFC3339),
		},
	})
}

func (h *AtCoderHandler) ListContestHistories(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	accountID, err := parseInt64Param(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	input, err := listPlatformSyncInputFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.ListContestHistories(c.Request.Context(), user.ID, accountID, input)
	if err != nil {
		writePlatformSyncError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.PlatformContestHistoriesResponse{
		Items: toPlatformContestHistoryResponses(items),
	})
}

func (h *AtCoderHandler) ListProblemFacts(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	input, err := listPlatformSyncInputFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.ListProblemFacts(c.Request.Context(), user.ID, input)
	if err != nil {
		writePlatformSyncError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.ProblemFactsResponse{
		Items: toProblemFactResponses(items),
	})
}

func (h *AtCoderHandler) ListContestSummaries(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	input, err := listPlatformSyncInputFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.ListContestSummaries(c.Request.Context(), user.ID, input)
	if err != nil {
		writePlatformSyncError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.ContestACSummariesResponse{
		Items: toContestACSummaryResponses(items),
	})
}
