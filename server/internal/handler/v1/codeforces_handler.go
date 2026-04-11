package v1

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	dtov1 "github.com/ICE-awa/acmrank/server/internal/dto/v1"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type CodeforcesSyncService interface {
	EnqueueSync(ctx context.Context, siteUserID int64, accountID int64) (model.SyncJob, error)
	GetLatestProfile(ctx context.Context, siteUserID int64, accountID int64) (model.PlatformProfileSnapshot, error)
	ListContestHistories(ctx context.Context, siteUserID int64, accountID int64, input service.ListPlatformSyncInput) ([]model.PlatformContestHistory, error)
	ListProblemFacts(ctx context.Context, siteUserID int64, input service.ListPlatformSyncInput) ([]model.ProblemFact, error)
	ListContestSummaries(ctx context.Context, siteUserID int64, input service.ListPlatformSyncInput) ([]model.ContestACSummary, error)
}

type CodeforcesHandler struct {
	service CodeforcesSyncService
}

func NewCodeforcesHandler(service CodeforcesSyncService) *CodeforcesHandler {
	return &CodeforcesHandler{service: service}
}

func (h *CodeforcesHandler) Sync(c *gin.Context) {
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
		writeCodeforcesError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job": toSyncJobResponse(job),
	})
}

func (h *CodeforcesHandler) GetLatestProfile(c *gin.Context) {
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
		writeCodeforcesError(c, err)
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

func (h *CodeforcesHandler) ListContestHistories(c *gin.Context) {
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
		writeCodeforcesError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.PlatformContestHistoriesResponse{
		Items: toPlatformContestHistoryResponses(items),
	})
}

func (h *CodeforcesHandler) ListProblemFacts(c *gin.Context) {
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
		writeCodeforcesError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.ProblemFactsResponse{
		Items: toProblemFactResponses(items),
	})
}

func (h *CodeforcesHandler) ListContestSummaries(c *gin.Context) {
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
		writeCodeforcesError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.ContestACSummariesResponse{
		Items: toContestACSummaryResponses(items),
	})
}

func listPlatformSyncInputFromQuery(c *gin.Context) (service.ListPlatformSyncInput, error) {
	limit, err := parseOptionalNonNegativeIntQuery(c, "limit")
	if err != nil {
		return service.ListPlatformSyncInput{}, err
	}

	offset, err := parseOptionalNonNegativeIntQuery(c, "offset")
	if err != nil {
		return service.ListPlatformSyncInput{}, err
	}

	return service.ListPlatformSyncInput{
		Limit:  limit,
		Offset: offset,
	}, nil
}

func toPlatformContestHistoryResponses(
	items []model.PlatformContestHistory,
) []dtov1.PlatformContestHistoryResponse {
	responses := make([]dtov1.PlatformContestHistoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dtov1.PlatformContestHistoryResponse{
			ID:             item.ID,
			ContestID:      item.ContestID,
			ContestName:    item.ContestName,
			Rank:           item.Rank,
			OldRating:      item.OldRating,
			NewRating:      item.NewRating,
			RatingDelta:    item.RatingDelta,
			ParticipatedAt: item.ParticipatedAt.UTC().Format(time.RFC3339),
			Source:         item.Source,
			SourceURL:      item.SourceURL,
			FetchedAt:      item.FetchedAt.UTC().Format(time.RFC3339),
			CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return responses
}

func toProblemFactResponses(items []model.ProblemFact) []dtov1.ProblemFactResponse {
	responses := make([]dtov1.ProblemFactResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dtov1.ProblemFactResponse{
			ID:                   item.ID,
			Platform:             string(item.Platform),
			ProblemKey:           item.ProblemKey,
			ContestID:            item.ContestID,
			ProblemIndexOrTaskID: item.ProblemIndexOrTaskID,
			ProblemName:          item.ProblemName,
			ProblemURL:           item.ProblemURL,
			FirstACAt:            item.FirstACAt.UTC().Format(time.RFC3339),
			FirstACSource:        item.FirstACSource,
			FirstACSubmissionRef: item.FirstACSubmissionRef,
			LatestACAt:           item.LatestACAt.UTC().Format(time.RFC3339),
			ClistRating:          item.ClistRating,
			CreatedAt:            item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:            item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return responses
}

func toContestACSummaryResponses(
	items []model.ContestACSummary,
) []dtov1.ContestACSummaryResponse {
	responses := make([]dtov1.ContestACSummaryResponse, 0, len(items))
	for _, item := range items {
		var firstACAt *string
		if item.FirstACAt != nil {
			formatted := item.FirstACAt.UTC().Format(time.RFC3339)
			firstACAt = &formatted
		}

		var lastACAt *string
		if item.LastACAt != nil {
			formatted := item.LastACAt.UTC().Format(time.RFC3339)
			lastACAt = &formatted
		}

		responses = append(responses, dtov1.ContestACSummaryResponse{
			ID:            item.ID,
			Platform:      string(item.Platform),
			ContestID:     item.ContestID,
			ContestName:   item.ContestName,
			ACProblemKeys: item.ACProblemKeys,
			ACCount:       item.ACCount,
			FirstACAt:     firstACAt,
			LastACAt:      lastACAt,
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return responses
}

func writeCodeforcesError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()
	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = http.StatusBadRequest
	case errors.Is(err, service.ErrPlatformAccountNotFound), errors.Is(err, service.ErrCodeforcesSyncDataNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, service.ErrPlatformAccountForbidden):
		statusCode = http.StatusForbidden
	case errors.Is(err, service.ErrPlatformAccountNotReady):
		statusCode = http.StatusConflict
	case errors.Is(err, service.ErrCodeforcesUpstream):
		statusCode = http.StatusBadGateway
	}

	if statusCode == http.StatusInternalServerError {
		log.Printf("codeforces handler error: %v", err)
		message = "internal server error"
	}

	c.JSON(statusCode, gin.H{"error": message})
}
