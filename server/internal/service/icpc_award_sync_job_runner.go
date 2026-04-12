package service

import (
	"context"
	"log"
	"time"
)

const defaultICPCAwardSyncPollInterval = time.Second

type ICPCAwardSyncJobProcessor interface {
	ProcessNextQueuedSync(context.Context) (bool, error)
}

type ICPCAwardSyncJobRunner struct {
	processor ICPCAwardSyncJobProcessor
	interval  time.Duration
}

func NewICPCAwardSyncJobRunner(
	processor ICPCAwardSyncJobProcessor,
	interval time.Duration,
) *ICPCAwardSyncJobRunner {
	if interval <= 0 {
		interval = defaultICPCAwardSyncPollInterval
	}

	return &ICPCAwardSyncJobRunner{
		processor: processor,
		interval:  interval,
	}
}

func (r *ICPCAwardSyncJobRunner) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *ICPCAwardSyncJobRunner) run(ctx context.Context) {
	r.drain(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.drain(ctx)
		}
	}
}

func (r *ICPCAwardSyncJobRunner) drain(ctx context.Context) {
	for {
		processed, err := r.processor.ProcessNextQueuedSync(ctx)
		if err != nil {
			log.Printf("icpc award sync runner error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}
