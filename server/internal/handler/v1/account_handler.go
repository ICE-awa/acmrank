package v1

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	dtov1 "github.com/ICE-awa/acmrank/server/internal/dto/v1"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type PlatformAccountService interface {
	ListMine(ctx context.Context, siteUserID int64, input service.ListPlatformAccountsInput) ([]model.PlatformAccount, error)
	Create(
		ctx context.Context,
		siteUserID int64,
		input service.CreatePlatformAccountInput,
	) (model.PlatformAccount, error)
	Delete(ctx context.Context, siteUserID int64, accountID int64) error
	ListAll(ctx context.Context, input service.ListPlatformAccountsInput) ([]model.PlatformAccount, error)
	Review(
		ctx context.Context,
		accountID int64,
		input service.ReviewPlatformAccountInput,
	) (model.PlatformAccount, error)
}

type PlatformAccountHandler struct {
	service PlatformAccountService
}

func NewPlatformAccountHandler(service PlatformAccountService) *PlatformAccountHandler {
	return &PlatformAccountHandler{service: service}
}

func (h *PlatformAccountHandler) ListMine(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	listInput, err := listPlatformAccountsInputFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accounts, err := h.service.ListMine(c.Request.Context(), user.ID, listInput)
	if err != nil {
		writePlatformAccountError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.PlatformAccountsResponse{
		Accounts: toPlatformAccountResponses(accounts, false),
	})
}

func (h *PlatformAccountHandler) Create(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var request dtov1.CreatePlatformAccountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.service.Create(c.Request.Context(), user.ID, service.CreatePlatformAccountInput{
		Platform: request.Platform,
		Handle:   request.Handle,
	})
	if err != nil {
		writePlatformAccountError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"account": toPlatformAccountResponse(account, false),
	})
}

func (h *PlatformAccountHandler) Delete(c *gin.Context) {
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

	if err := h.service.Delete(c.Request.Context(), user.ID, accountID); err != nil {
		writePlatformAccountError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *PlatformAccountHandler) ListAll(c *gin.Context) {
	listInput, err := listPlatformAccountsInputFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accounts, err := h.service.ListAll(c.Request.Context(), listInput)
	if err != nil {
		writePlatformAccountError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.PlatformAccountsResponse{
		Accounts: toPlatformAccountResponses(accounts, true),
	})
}

func (h *PlatformAccountHandler) Verify(c *gin.Context) {
	h.review(c, model.PlatformAccountStatusVerified)
}

func (h *PlatformAccountHandler) Disable(c *gin.Context) {
	h.review(c, model.PlatformAccountStatusDisabled)
}

func (h *PlatformAccountHandler) Reject(c *gin.Context) {
	h.review(c, model.PlatformAccountStatusRejected)
}

func (h *PlatformAccountHandler) review(c *gin.Context, status model.PlatformAccountStatus) {
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

	var request dtov1.ReviewPlatformAccountRequest
	if err := bindOptionalJSON(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.service.Review(c.Request.Context(), accountID, service.ReviewPlatformAccountInput{
		Status:         string(status),
		Reason:         request.Reason,
		ReviewerUserID: user.ID,
	})
	if err != nil {
		writePlatformAccountError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account": toPlatformAccountResponse(account, true),
	})
}

func toPlatformAccountResponses(
	accounts []model.PlatformAccount,
	includeOwner bool,
) []dtov1.PlatformAccountResponse {
	responses := make([]dtov1.PlatformAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, toPlatformAccountResponse(account, includeOwner))
	}

	return responses
}

func toPlatformAccountResponse(
	account model.PlatformAccount,
	includeOwner bool,
) dtov1.PlatformAccountResponse {
	var verifiedAt *string
	if account.VerifiedAt != nil {
		formatted := account.VerifiedAt.UTC().Format(time.RFC3339)
		verifiedAt = &formatted
	}

	var lastSyncedAt *string
	if account.LastSyncedAt != nil {
		formatted := account.LastSyncedAt.UTC().Format(time.RFC3339)
		lastSyncedAt = &formatted
	}

	var owner *dtov1.PlatformAccountOwnerResponse
	if includeOwner {
		owner = &dtov1.PlatformAccountOwnerResponse{
			ID:       account.Owner.ID,
			Username: account.Owner.Username,
			RealName: account.Owner.RealName,
		}
	}

	return dtov1.PlatformAccountResponse{
		ID:            account.ID,
		Platform:      string(account.Platform),
		Handle:        account.Handle,
		DisplayHandle: account.DisplayHandle,
		Status:        string(account.Status),
		VerifiedAt:    verifiedAt,
		LastSyncedAt:  lastSyncedAt,
		CreatedAt:     account.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     account.UpdatedAt.UTC().Format(time.RFC3339),
		Owner:         owner,
	}
}

func writePlatformAccountError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()
	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = http.StatusBadRequest
	case errors.Is(err, service.ErrPlatformAccountUnavailable):
		statusCode = http.StatusConflict
	case errors.Is(err, service.ErrPlatformAccountNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, service.ErrPlatformAccountForbidden):
		statusCode = http.StatusForbidden
	}

	if statusCode == http.StatusInternalServerError {
		log.Printf("platform account handler error: %v", err)
		message = "internal server error"
	}

	c.JSON(statusCode, gin.H{"error": message})
}

func parseInt64Param(c *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func listPlatformAccountsInputFromQuery(c *gin.Context) (service.ListPlatformAccountsInput, error) {
	limit, err := parseOptionalNonNegativeIntQuery(c, "limit")
	if err != nil {
		return service.ListPlatformAccountsInput{}, err
	}

	offset, err := parseOptionalNonNegativeIntQuery(c, "offset")
	if err != nil {
		return service.ListPlatformAccountsInput{}, err
	}

	return service.ListPlatformAccountsInput{
		Platform: c.Query("platform"),
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func parseOptionalNonNegativeIntQuery(c *gin.Context, name string) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(name + " must be a non-negative integer")
	}
	if value < 0 {
		return 0, errors.New(name + " must be a non-negative integer")
	}

	return value, nil
}

func bindOptionalJSON(c *gin.Context, target any) error {
	if c.Request.Body == nil {
		return nil
	}

	err := c.ShouldBindJSON(target)
	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}
