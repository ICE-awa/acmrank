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

type LuoguSyncService interface {
	GetLatestProfile(ctx context.Context, siteUserID int64, accountID int64) (model.PlatformProfileSnapshot, error)
	ListProblemFacts(ctx context.Context, siteUserID int64, input service.ListCodeforcesSyncInput) ([]model.ProblemFact, error)
}

type LuoguHandler struct {
	service LuoguSyncService
}

func NewLuoguHandler(service LuoguSyncService) *LuoguHandler {
	return &LuoguHandler{service: service}
}

func (h *LuoguHandler) GetLatestProfile(c *gin.Context) {
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

func (h *LuoguHandler) ListProblemFacts(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	input, err := listCodeforcesSyncInputFromQuery(c)
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
