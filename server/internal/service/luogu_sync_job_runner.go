package service

import (
	"context"
	"log"
	"time"
)

const defaultLuoguSyncPollInterval = time.Second

type LuoguSyncJobProcessor interface {
	ProcessNextQueuedSync(context.Context) (bool, error)
}

type LuoguSyncJobRunner struct {
	processor LuoguSyncJobProcessor
	interval  time.Duration
}

func NewLuoguSyncJobRunner(
	processor LuoguSyncJobProcessor,
	interval time.Duration,
) *LuoguSyncJobRunner {
	if interval <= 0 {
		interval = defaultLuoguSyncPollInterval
	}

	return &LuoguSyncJobRunner{
		processor: processor,
		interval:  interval,
	}
}

func (r *LuoguSyncJobRunner) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *LuoguSyncJobRunner) run(ctx context.Context) {
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

func (r *LuoguSyncJobRunner) drain(ctx context.Context) {
	for {
		processed, err := r.processor.ProcessNextQueuedSync(ctx)
		if err != nil {
			log.Printf("luogu sync runner error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}
