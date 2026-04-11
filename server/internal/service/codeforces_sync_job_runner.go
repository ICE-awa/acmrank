package service

import (
	"context"
	"log"
	"time"
)

const defaultCodeforcesSyncPollInterval = time.Second

type CodeforcesSyncJobProcessor interface {
	ProcessNextQueuedSync(context.Context) (bool, error)
}

type CodeforcesSyncJobRunner struct {
	processor CodeforcesSyncJobProcessor
	interval  time.Duration
}

func NewCodeforcesSyncJobRunner(
	processor CodeforcesSyncJobProcessor,
	interval time.Duration,
) *CodeforcesSyncJobRunner {
	if interval <= 0 {
		interval = defaultCodeforcesSyncPollInterval
	}

	return &CodeforcesSyncJobRunner{
		processor: processor,
		interval:  interval,
	}
}

func (r *CodeforcesSyncJobRunner) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *CodeforcesSyncJobRunner) run(ctx context.Context) {
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

func (r *CodeforcesSyncJobRunner) drain(ctx context.Context) {
	for {
		processed, err := r.processor.ProcessNextQueuedSync(ctx)
		if err != nil {
			log.Printf("codeforces sync runner error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}
