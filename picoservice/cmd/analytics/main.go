package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"picoservice/internal/app"
	"picoservice/internal/dao/mongodb"
)

const jobTimeout = 10 * time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("link analytics job stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := app.AnalyticsConfigFromEnv()
	if err != nil {
		return err
	}

	db, err := mongodb.OpenFromEnv()
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Close(ctx); err != nil {
			logger.Error("close mongodb", "error", err)
		}
	}()

	job, err := app.NewLinkAnalyticsJob(db, cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	result, err := job.Run(ctx)
	if err != nil {
		return err
	}
	logger.Info("link analytics job completed",
		"skipped", result.Skipped,
		"current_hour", result.CurrentHourProcessed,
		"previous_hour", result.PreviousHourProcessed,
		"hours_processed", result.HoursProcessed,
		"gap_hours", result.GapHours,
		"clicks", result.Clicks,
	)
	return nil
}
