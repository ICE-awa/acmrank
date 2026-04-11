package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/repository"
)

type stubPlatformAccountStore struct {
	createFn       func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error)
	listByUserFn   func(context.Context, int64) ([]model.PlatformAccount, error)
	listFn         func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error)
	getByIDFn      func(context.Context, int64) (model.PlatformAccount, error)
	deleteByUserFn func(context.Context, int64, int64) error
	reviewFn       func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error)
}

func (s stubPlatformAccountStore) Create(
	ctx context.Context,
	params repository.CreatePlatformAccountParams,
) (model.PlatformAccount, error) {
	return s.createFn(ctx, params)
}

func (s stubPlatformAccountStore) ListByUserID(
	ctx context.Context,
	siteUserID int64,
) ([]model.PlatformAccount, error) {
	return s.listByUserFn(ctx, siteUserID)
}

func (s stubPlatformAccountStore) List(
	ctx context.Context,
	filter repository.ListPlatformAccountsFilter,
) ([]model.PlatformAccount, error) {
	return s.listFn(ctx, filter)
}

func (s stubPlatformAccountStore) GetByID(
	ctx context.Context,
	accountID int64,
) (model.PlatformAccount, error) {
	return s.getByIDFn(ctx, accountID)
}

func (s stubPlatformAccountStore) DeleteByUserID(
	ctx context.Context,
	accountID int64,
	siteUserID int64,
) error {
	return s.deleteByUserFn(ctx, accountID, siteUserID)
}

func (s stubPlatformAccountStore) Review(
	ctx context.Context,
	params repository.ReviewPlatformAccountParams,
) (model.PlatformAccount, error) {
	return s.reviewFn(ctx, params)
}

func TestPlatformAccountServiceCreateNormalizesInput(t *testing.T) {
	t.Parallel()

	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(_ context.Context, params repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			if params.Platform != model.PlatformAtCoder || params.Handle != "tourist" {
				t.Fatalf("Create() params = %+v", params)
			}

			return model.PlatformAccount{
				ID:            1,
				SiteUserID:    7,
				Platform:      params.Platform,
				Handle:        params.Handle,
				DisplayHandle: params.DisplayHandle,
				Status:        model.PlatformAccountStatusPendingReview,
			}, nil
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn:      func(context.Context, int64) (model.PlatformAccount, error) { return model.PlatformAccount{}, nil },
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})

	account, err := service.Create(context.Background(), 7, CreatePlatformAccountInput{
		Platform: " AtCoder ",
		Handle:   " tourist ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if account.DisplayHandle != "tourist" {
		t.Fatalf("Create() display handle = %q, want %q", account.DisplayHandle, "tourist")
	}
}

func TestPlatformAccountServiceCreateMapsUniquenessError(t *testing.T) {
	t.Parallel()

	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, repository.ErrPlatformAccountAlreadyBound
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn:      func(context.Context, int64) (model.PlatformAccount, error) { return model.PlatformAccount{}, nil },
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})

	_, err := service.Create(context.Background(), 1, CreatePlatformAccountInput{
		Platform: "codeforces",
		Handle:   "neal",
	})
	if !errors.Is(err, ErrPlatformAccountUnavailable) {
		t.Fatalf("Create() error = %v, want %v", err, ErrPlatformAccountUnavailable)
	}
}

func TestPlatformAccountServiceDeleteRejectsForeignAccount(t *testing.T) {
	t.Parallel()

	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
			return model.PlatformAccount{ID: 9, SiteUserID: 22}, nil
		},
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})

	err := service.Delete(context.Background(), 7, 9)
	if !errors.Is(err, ErrPlatformAccountForbidden) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrPlatformAccountForbidden)
	}
}

func TestPlatformAccountServiceListAllValidatesFilter(t *testing.T) {
	t.Parallel()

	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn:      func(context.Context, int64) (model.PlatformAccount, error) { return model.PlatformAccount{}, nil },
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
	})

	_, err := service.ListAll(context.Background(), ListPlatformAccountsInput{Status: "unknown"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ListAll() error = %v, want validation error", err)
	}
}

func TestPlatformAccountServiceReviewIsIdempotentForSameStatus(t *testing.T) {
	t.Parallel()

	reviewCalls := 0
	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
			return model.PlatformAccount{
				ID:     5,
				Status: model.PlatformAccountStatusVerified,
			}, nil
		},
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(context.Context, repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			reviewCalls++
			return model.PlatformAccount{}, nil
		},
	})

	account, err := service.Review(context.Background(), 5, ReviewPlatformAccountInput{
		Status:         "verified",
		ReviewerUserID: 1,
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if reviewCalls != 0 {
		t.Fatalf("Review() reviewCalls = %d, want 0", reviewCalls)
	}

	if account.Status != model.PlatformAccountStatusVerified {
		t.Fatalf("Review() status = %q, want %q", account.Status, model.PlatformAccountStatusVerified)
	}
}

func TestPlatformAccountServiceReviewPassesReviewerAndTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_100_300, 0).UTC()
	service := NewPlatformAccountService(stubPlatformAccountStore{
		createFn: func(context.Context, repository.CreatePlatformAccountParams) (model.PlatformAccount, error) {
			return model.PlatformAccount{}, nil
		},
		listByUserFn: func(context.Context, int64) ([]model.PlatformAccount, error) { return nil, nil },
		listFn: func(context.Context, repository.ListPlatformAccountsFilter) ([]model.PlatformAccount, error) {
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (model.PlatformAccount, error) {
			return model.PlatformAccount{
				ID:     8,
				Status: model.PlatformAccountStatusPendingReview,
			}, nil
		},
		deleteByUserFn: func(context.Context, int64, int64) error { return nil },
		reviewFn: func(_ context.Context, params repository.ReviewPlatformAccountParams) (model.PlatformAccount, error) {
			if params.ReviewerUserID != 3 {
				t.Fatalf("Review() reviewer = %d, want %d", params.ReviewerUserID, 3)
			}

			if !params.ReviewedAt.Equal(now) {
				t.Fatalf("Review() reviewed at = %v, want %v", params.ReviewedAt, now)
			}

			return model.PlatformAccount{
				ID:     8,
				Status: model.PlatformAccountStatusRejected,
			}, nil
		},
	})
	service.now = func() time.Time { return now }

	account, err := service.Review(context.Background(), 8, ReviewPlatformAccountInput{
		Status:         "rejected",
		Reason:         "ownership mismatch",
		ReviewerUserID: 3,
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if account.Status != model.PlatformAccountStatusRejected {
		t.Fatalf("Review() status = %q, want %q", account.Status, model.PlatformAccountStatusRejected)
	}
}
