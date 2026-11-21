package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"picoservice/internal/app"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/lambdahttp"
)

const closeTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_MODE")), "analytics") {
		runAnalytics(logger)
		return
	}
	runAPI(logger)
}

func runAPI(logger *slog.Logger) {
	cfg, err := app.ConfigFromEnv()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := mongodb.OpenFromEnv()
	if err != nil {
		logger.Error("open mongodb", "error", err)
		os.Exit(1)
	}
	defer closeDatabase(logger, db)

	handler, err := app.New(logger, db, cfg)
	if err != nil {
		logger.Error("build application", "error", err)
		os.Exit(1)
	}

	adapter := lambdahttp.New(handler, logger)
	lambda.Start(adapter.Handle)
}

func runAnalytics(logger *slog.Logger) {
	cfg, err := app.AnalyticsConfigFromEnv()
	if err != nil {
		logger.Error("load analytics config", "error", err)
		os.Exit(1)
	}

	db, err := mongodb.OpenFromEnv()
	if err != nil {
		logger.Error("open mongodb", "error", err)
		os.Exit(1)
	}
	defer closeDatabase(logger, db)

	job, err := app.NewLinkAnalyticsJob(db, cfg)
	if err != nil {
		logger.Error("build link analytics job", "error", err)
		os.Exit(1)
	}

	lambda.Start(func(ctx context.Context, _ json.RawMessage) error {
		result, err := job.Run(ctx)
		if err != nil {
			logger.Error("link analytics job failed", "error", err)
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
	})
}

func closeDatabase(logger *slog.Logger, db *mongodb.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()

	if err := db.Close(ctx); err != nil {
		logger.Error("close mongodb", "error", err)
	}
}
