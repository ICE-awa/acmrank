package service

import (
	"context"
	"log"
	"time"
)

const defaultAtCoderSyncPollInterval = time.Second

type AtCoderSyncJobProcessor interface {
	ProcessNextQueuedSync(context.Context) (bool, error)
}

type AtCoderSyncJobRunner struct {
	processor AtCoderSyncJobProcessor
	interval  time.Duration
}

func NewAtCoderSyncJobRunner(
	processor AtCoderSyncJobProcessor,
	interval time.Duration,
) *AtCoderSyncJobRunner {
	if interval <= 0 {
		interval = defaultAtCoderSyncPollInterval
	}

	return &AtCoderSyncJobRunner{
		processor: processor,
		interval:  interval,
	}
}

func (r *AtCoderSyncJobRunner) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *AtCoderSyncJobRunner) run(ctx context.Context) {
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

func (r *AtCoderSyncJobRunner) drain(ctx context.Context) {
	for {
		processed, err := r.processor.ProcessNextQueuedSync(ctx)
		if err != nil {
			log.Printf("atcoder sync runner error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}
