package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

var (
	ErrPlatformAccountNotFound    = errors.New("platform account not found")
	ErrPlatformAccountUnavailable = errors.New("platform account already bound")
	ErrPlatformAccountForbidden   = errors.New("platform account access forbidden")
)

const (
	defaultPlatformAccountListLimit = 50
	maxPlatformAccountListLimit     = 100
)

type PlatformAccountStore interface {
	Create(ctx context.Context, params repository.CreatePlatformAccountParams) (model.PlatformAccount, error)
	ListByUserID(ctx context.Context, siteUserID int64, filter repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error)
	List(ctx context.Context, filter repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error)
	GetByID(ctx context.Context, accountID int64) (model.PlatformAccount, error)
	DeleteByUserID(ctx context.Context, accountID int64, siteUserID int64) error
	Review(ctx context.Context, params repository.ReviewPlatformAccountParams) (model.PlatformAccount, error)
}

type CreatePlatformAccountInput struct {
	Platform string
	Handle   string
}

type ListPlatformAccountsInput struct {
	Platform string
	Status   string
	Limit    int
	Offset   int
}

type ReviewPlatformAccountInput struct {
	Status         string
	Reason         string
	ReviewerUserID int64
}

type PlatformAccountService struct {
	store PlatformAccountStore
	now   func() time.Time
}

func NewPlatformAccountService(store PlatformAccountStore) *PlatformAccountService {
	return &PlatformAccountService{
		store: store,
		now:   time.Now,
	}
}

func (s *PlatformAccountService) ListMine(
	ctx context.Context,
	siteUserID int64,
	input ListPlatformAccountsInput,
) ([]model.PlatformAccount, error) {
	filter, err := normalizeListPlatformAccountsInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.ListByUserID(ctx, siteUserID, filter)
}

func (s *PlatformAccountService) Create(
	ctx context.Context,
	siteUserID int64,
	input CreatePlatformAccountInput,
) (model.PlatformAccount, error) {
	normalized, err := normalizeCreatePlatformAccountInput(input)
	if err != nil {
		return model.PlatformAccount{}, err
	}

	account, err := s.store.Create(ctx, repository.CreatePlatformAccountParams{
		SiteUserID:    siteUserID,
		Platform:      normalized.Platform,
		Handle:        normalized.Handle,
		DisplayHandle: normalized.Handle,
	})
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountStoreError(err)
	}

	return account, nil
}

func (s *PlatformAccountService) Delete(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) error {
	account, err := s.store.GetByID(ctx, accountID)
	if err != nil {
		return mapPlatformAccountStoreError(err)
	}

	if account.SiteUserID != siteUserID {
		return ErrPlatformAccountForbidden
	}

	if err := s.store.DeleteByUserID(ctx, accountID, siteUserID); err != nil {
		return mapPlatformAccountStoreError(err)
	}

	return nil
}

func (s *PlatformAccountService) ListAll(
	ctx context.Context,
	input ListPlatformAccountsInput,
) ([]model.PlatformAccount, error) {
	filter, err := normalizeListPlatformAccountsInput(input)
	if err != nil {
		return nil, err
	}

	accounts, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (s *PlatformAccountService) Review(
	ctx context.Context,
	accountID int64,
	input ReviewPlatformAccountInput,
) (model.PlatformAccount, error) {
	normalized, err := normalizeReviewPlatformAccountInput(input)
	if err != nil {
		return model.PlatformAccount{}, err
	}

	account, err := s.store.GetByID(ctx, accountID)
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountStoreError(err)
	}

	if account.Status == normalized.Status {
		return account, nil
	}

	updated, err := s.store.Review(ctx, repository.ReviewPlatformAccountParams{
		AccountID:      accountID,
		ReviewerUserID: normalized.ReviewerUserID,
		Status:         normalized.Status,
		Reason:         normalized.Reason,
		ReviewedAt:     s.now().UTC(),
	})
	if err != nil {
		return model.PlatformAccount{}, mapPlatformAccountStoreError(err)
	}

	return updated, nil
}

type normalizedCreatePlatformAccountInput struct {
	Platform model.Platform
	Handle   string
}

func normalizeCreatePlatformAccountInput(
	input CreatePlatformAccountInput,
) (normalizedCreatePlatformAccountInput, error) {
	platform := model.Platform(strings.ToLower(strings.TrimSpace(input.Platform)))
	handle := strings.TrimSpace(input.Handle)

	switch {
	case !platform.IsSupported():
		return normalizedCreatePlatformAccountInput{}, ValidationError{
			Message: "platform must be one of codeforces, atcoder, or luogu",
		}
	case handle == "":
		return normalizedCreatePlatformAccountInput{}, ValidationError{
			Message: "handle is required",
		}
	case len([]rune(handle)) > 64:
		return normalizedCreatePlatformAccountInput{}, ValidationError{
			Message: "handle must be 64 characters or fewer",
		}
	default:
		return normalizedCreatePlatformAccountInput{
			Platform: platform,
			Handle:   handle,
		}, nil
	}
}

func normalizeListPlatformAccountsInput(
	input ListPlatformAccountsInput,
) (repository.ListPlatformAccountsFilter, error) {
	filter := repository.ListPlatformAccountsFilter{}

	if trimmedPlatform := strings.TrimSpace(input.Platform); trimmedPlatform != "" {
		platform := model.Platform(strings.ToLower(trimmedPlatform))
		if !platform.IsSupported() {
			return repository.ListPlatformAccountsFilter{}, ValidationError{
				Message: "platform must be one of codeforces, atcoder, or luogu",
			}
		}

		filter.Platform = platform
	}

	if trimmedStatus := strings.TrimSpace(input.Status); trimmedStatus != "" {
		status := model.PlatformAccountStatus(strings.ToLower(trimmedStatus))
		if !status.IsSupported() {
			return repository.ListPlatformAccountsFilter{}, ValidationError{
				Message: "status must be one of pending_review, verified, disabled, or rejected",
			}
		}

		filter.Status = status
	}

	switch {
	case input.Limit < 0:
		return repository.ListPlatformAccountsFilter{}, ValidationError{
			Message: "limit must be greater than or equal to 0",
		}
	case input.Offset < 0:
		return repository.ListPlatformAccountsFilter{}, ValidationError{
			Message: "offset must be greater than or equal to 0",
		}
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultPlatformAccountListLimit
	}
	if limit > maxPlatformAccountListLimit {
		limit = maxPlatformAccountListLimit
	}

	filter.Limit = limit
	filter.Offset = input.Offset

	return filter, nil
}

type normalizedReviewPlatformAccountInput struct {
	Status         model.PlatformAccountStatus
	Reason         string
	ReviewerUserID int64
}

func normalizeReviewPlatformAccountInput(
	input ReviewPlatformAccountInput,
) (normalizedReviewPlatformAccountInput, error) {
	status := model.PlatformAccountStatus(strings.ToLower(strings.TrimSpace(input.Status)))
	reason := strings.TrimSpace(input.Reason)

	switch {
	case input.ReviewerUserID <= 0:
		return normalizedReviewPlatformAccountInput{}, ValidationError{
			Message: "reviewer is required",
		}
	case !status.IsSupported():
		return normalizedReviewPlatformAccountInput{}, ValidationError{
			Message: "status must be one of pending_review, verified, disabled, or rejected",
		}
	case len([]rune(reason)) > 256:
		return normalizedReviewPlatformAccountInput{}, ValidationError{
			Message: "reason must be 256 characters or fewer",
		}
	default:
		return normalizedReviewPlatformAccountInput{
			Status:         status,
			Reason:         reason,
			ReviewerUserID: input.ReviewerUserID,
		}, nil
	}
}

func mapPlatformAccountStoreError(err error) error {
	switch {
	case errors.Is(err, repository.ErrPlatformAccountNotFound):
		return ErrPlatformAccountNotFound
	case errors.Is(err, repository.ErrPlatformAccountAlreadyBound):
		return ErrPlatformAccountUnavailable
	default:
		return fmt.Errorf("platform account store: %w", err)
	}
}
