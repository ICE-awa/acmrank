package service

import (
	"context"

	"github.com/ICE-awa/acmrank/server/internal/model"
)

type PlatformSyncAccountStore interface {
	GetByID(ctx context.Context, accountID int64) (model.PlatformAccount, error)
}

type PlatformSyncEnqueuer interface {
	EnqueueSync(ctx context.Context, siteUserID int64, accountID int64) (model.SyncJob, error)
}

type PlatformSyncService struct {
	accountStore     PlatformSyncAccountStore
	atcoderSyncer    PlatformSyncEnqueuer
	codeforcesSyncer PlatformSyncEnqueuer
	luoguSyncer      PlatformSyncEnqueuer
}

func NewPlatformSyncService(
	accountStore PlatformSyncAccountStore,
	atcoderSyncer PlatformSyncEnqueuer,
	codeforcesSyncer PlatformSyncEnqueuer,
	luoguSyncer PlatformSyncEnqueuer,
) *PlatformSyncService {
	return &PlatformSyncService{
		accountStore:     accountStore,
		atcoderSyncer:    atcoderSyncer,
		codeforcesSyncer: codeforcesSyncer,
		luoguSyncer:      luoguSyncer,
	}
}

func (s *PlatformSyncService) EnqueueSync(
	ctx context.Context,
	siteUserID int64,
	accountID int64,
) (model.SyncJob, error) {
	account, err := s.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return model.SyncJob{}, mapPlatformAccountStoreError(err)
	}

	if account.SiteUserID != siteUserID {
		return model.SyncJob{}, ErrPlatformAccountForbidden
	}

	switch account.Platform {
	case model.PlatformAtCoder:
		return s.atcoderSyncer.EnqueueSync(ctx, siteUserID, accountID)
	case model.PlatformCodeforces:
		return s.codeforcesSyncer.EnqueueSync(ctx, siteUserID, accountID)
	case model.PlatformLuogu:
		return s.luoguSyncer.EnqueueSync(ctx, siteUserID, accountID)
	default:
		return model.SyncJob{}, ValidationError{Message: "sync is not supported for this platform yet"}
	}
}
