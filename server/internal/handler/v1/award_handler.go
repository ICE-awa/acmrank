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

type AwardService interface {
	EnqueueSync(ctx context.Context, siteUserID int64) (model.SyncJob, error)
	ListAwards(ctx context.Context, siteUserID int64, input service.ListAwardRecordsInput) ([]model.AwardRecord, error)
}

type AwardHandler struct {
	service AwardService
}

func NewAwardHandler(service AwardService) *AwardHandler {
	return &AwardHandler{service: service}
}

func (h *AwardHandler) Sync(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	job, err := h.service.EnqueueSync(c.Request.Context(), user.ID)
	if err != nil {
		writeAwardError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job": toSyncJobResponse(job),
	})
}

func (h *AwardHandler) ListMine(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	limit, err := parseOptionalNonNegativeIntQuery(c, "limit")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	offset, err := parseOptionalNonNegativeIntQuery(c, "offset")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.ListAwards(c.Request.Context(), user.ID, service.ListAwardRecordsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeAwardError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.AwardRecordsResponse{
		Items: toAwardRecordResponses(items),
	})
}

func toAwardRecordResponses(items []model.AwardRecord) []dtov1.AwardRecordResponse {
	responses := make([]dtov1.AwardRecordResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dtov1.AwardRecordResponse{
			ID:          item.ID,
			Platform:    item.Platform,
			ContestName: item.ContestName,
			AwardName:   item.AwardName,
			RankText:    item.RankText,
			AwardDate:   item.AwardDate.UTC().Format(time.DateOnly),
			Source:      item.Source,
			SourceURL:   item.SourceURL,
			IsManual:    item.IsManual,
			Notes:       item.Notes,
			CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return responses
}

func writeAwardError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = http.StatusBadRequest
	case errors.Is(err, service.ErrICPCAwardUpstream):
		statusCode = http.StatusBadGateway
		message = "upstream service unavailable"
	}

	if statusCode == http.StatusInternalServerError {
		log.Printf("award handler error: %v", err)
		message = "internal server error"
	}

	c.JSON(statusCode, gin.H{"error": message})
}
