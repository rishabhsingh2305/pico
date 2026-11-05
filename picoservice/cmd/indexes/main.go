package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"picoservice/internal/dao/mongodb"
)

const (
	indexTimeout = 30 * time.Second
	closeTimeout = 5 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("ensure indexes", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	client, err := mongodb.OpenFromEnv()
	if err != nil {
		return fmt.Errorf("open mongodb: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if err := client.Close(ctx); err != nil {
			logger.Error("close mongodb", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), indexTimeout)
	defer cancel()

	if err := client.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("ensure indexes: %w", err)
	}

	logger.Info("indexes ensured")
	return nil
}
