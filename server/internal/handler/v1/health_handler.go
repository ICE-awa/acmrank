package v1

import (
	"context"
	"net/http"
	"time"

	dtov1 "github.com/ICE-awa/acmrank/server/internal/dto/v1"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/gin-gonic/gin"
)

type HealthService interface {
	GetReport(ctx context.Context) (model.HealthReport, error)
}

type HealthHandler struct {
	service HealthService
}

func NewHealthHandler(service HealthService) *HealthHandler {
	return &HealthHandler{service: service}
}

func (h *HealthHandler) Get(c *gin.Context) {
	report, err := h.service.GetReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusCode := http.StatusOK
	if report.Status != model.HealthStatusOK {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, toHealthResponse(report))
}

func toHealthResponse(report model.HealthReport) dtov1.HealthResponse {
	dependencies := make([]dtov1.DependencyHealthResponse, 0, len(report.Dependencies))
	for _, dependency := range report.Dependencies {
		dependencies = append(dependencies, dtov1.DependencyHealthResponse{
			Name:       dependency.Name,
			Configured: dependency.Configured,
			Reachable:  dependency.Reachable,
			Message:    dependency.Message,
		})
	}

	return dtov1.HealthResponse{
		Service:      report.Service,
		Version:      report.Version,
		Status:       string(report.Status),
		StartedAt:    report.StartedAt.UTC().Format(time.RFC3339),
		CheckedAt:    report.CheckedAt.UTC().Format(time.RFC3339),
		Dependencies: dependencies,
	}
}
