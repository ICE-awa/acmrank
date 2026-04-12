package service

import (
	"context"
	"errors"
	"testing"
)

type stubICPCAwardRunnerProcessor struct {
	processFn func(context.Context) (bool, error)
}

func (s stubICPCAwardRunnerProcessor) ProcessNextQueuedSync(
	ctx context.Context,
) (bool, error) {
	return s.processFn(ctx)
}

func TestICPCAwardSyncJobRunnerDrainContinuesAfterProcessedError(t *testing.T) {
	t.Parallel()

	callCount := 0
	runner := NewICPCAwardSyncJobRunner(
		stubICPCAwardRunnerProcessor{
			processFn: func(context.Context) (bool, error) {
				callCount++
				switch callCount {
				case 1:
					return true, errors.New("first job failed")
				case 2:
					return false, nil
				default:
					t.Fatalf("unexpected process call %d", callCount)
					return false, nil
				}
			},
		},
		0,
	)

	runner.drain(context.Background())

	if callCount != 2 {
		t.Fatalf("drain() callCount = %d, want %d", callCount, 2)
	}
}

func TestICPCAwardSyncJobRunnerDrainStopsOnFatalError(t *testing.T) {
	t.Parallel()

	callCount := 0
	runner := NewICPCAwardSyncJobRunner(
		stubICPCAwardRunnerProcessor{
			processFn: func(context.Context) (bool, error) {
				callCount++
				return false, errors.New("claim failed")
			},
		},
		0,
	)

	runner.drain(context.Background())

	if callCount != 1 {
		t.Fatalf("drain() callCount = %d, want %d", callCount, 1)
	}
}
