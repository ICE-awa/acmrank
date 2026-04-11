package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := bootstrap.RunService(ctx, appmeta.ServiceAggregator); err != nil {
		log.Fatalf("aggregator service exited with error: %v", err)
	}
}
