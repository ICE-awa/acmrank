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

type PlatformSyncService interface {
	EnqueueSync(ctx context.Context, siteUserID int64, accountID int64) (model.SyncJob, error)
}

type PlatformSyncHandler struct {
	service PlatformSyncService
}

func NewPlatformSyncHandler(service PlatformSyncService) *PlatformSyncHandler {
	return &PlatformSyncHandler{service: service}
}

func (h *PlatformSyncHandler) Sync(c *gin.Context) {
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

func writePlatformSyncError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = http.StatusBadRequest
	case errors.Is(err, service.ErrPlatformAccountNotFound),
		errors.Is(err, service.ErrCodeforcesSyncDataNotFound),
		errors.Is(err, service.ErrLuoguSyncDataNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, service.ErrPlatformAccountForbidden):
		statusCode = http.StatusForbidden
	case errors.Is(err, service.ErrPlatformAccountNotReady):
		statusCode = http.StatusConflict
	case errors.Is(err, service.ErrCodeforcesUpstream),
		errors.Is(err, service.ErrLuoguUpstream):
		statusCode = http.StatusBadGateway
		message = "upstream service unavailable"
	}

	if statusCode == http.StatusInternalServerError {
		log.Printf("platform sync handler error: %v", err)
		message = "internal server error"
	}

	c.JSON(statusCode, gin.H{"error": message})
}

func toSyncJobResponse(job model.SyncJob) dtov1.SyncJobResponse {
	var startedAt *string
	if job.StartedAt != nil {
		formatted := job.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &formatted
	}

	var finishedAt *string
	if job.FinishedAt != nil {
		formatted := job.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &formatted
	}

	return dtov1.SyncJobResponse{
		ID:                job.ID,
		PlatformAccountID: job.PlatformAccountID,
		Platform:          job.Platform,
		JobType:           string(job.JobType),
		Status:            string(job.Status),
		ScheduledAt:       job.ScheduledAt.UTC().Format(time.RFC3339),
		StartedAt:         startedAt,
		FinishedAt:        finishedAt,
		AttemptCount:      job.AttemptCount,
		ErrorMessage:      job.ErrorMessage,
		CreatedAt:         job.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         job.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
