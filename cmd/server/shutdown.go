package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// setupGracefulShutdown sets up signal handling for graceful telemetry shutdown.
// It starts a goroutine that listens for interrupt signals and properly shuts down
// the telemetry provider with a timeout.
func setupGracefulShutdown(telemetryProvider TelemetryProvider, log *zap.Logger) {
	if telemetryProvider == nil {
		return
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down telemetry...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := telemetryProvider.Shutdown(ctx); err != nil {
			log.Error("Error shutting down telemetry", zap.Error(err))
		}
	}()
}
