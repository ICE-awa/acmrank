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

type stubHealthService struct {
	report model.HealthReport
	err    error
}

func (s stubHealthService) GetReport(context.Context) (model.HealthReport, error) {
	return s.report, s.err
}

func TestHealthHandlerReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewHealthHandler(stubHealthService{
		report: model.HealthReport{
			Service:   "api",
			Version:   "dev",
			Status:    model.HealthStatusOK,
			StartedAt: time.Unix(100, 0),
			CheckedAt: time.Unix(200, 0),
		},
	})
	router.GET("/healthz", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthHandlerReturnsServiceUnavailableForDegradedState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewHealthHandler(stubHealthService{
		report: model.HealthReport{
			Service:   "scheduler",
			Version:   "dev",
			Status:    model.HealthStatusDegraded,
			StartedAt: time.Unix(100, 0),
			CheckedAt: time.Unix(200, 0),
		},
	})
	router.GET("/healthz", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
