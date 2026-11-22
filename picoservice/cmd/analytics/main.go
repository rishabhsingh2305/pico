package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"picoservice/internal/app"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/services"
)

const jobTimeout = 10 * time.Minute

func main() {
	from := flag.String("from", "", "backfill start time (RFC 3339, UTC)")
	to := flag.String("to", "", "backfill end time (RFC 3339, UTC)")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, *from, *to); err != nil {
		logger.Error("link analytics job stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, from, to string) error {
	if (from == "") != (to == "") {
		return fmt.Errorf("--from and --to must be provided together")
	}
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

	mode := "run"
	var result services.LinkAnalyticsJobResult
	if from != "" {
		fromTime, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return fmt.Errorf("parse --from: %w", err)
		}
		toTime, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return fmt.Errorf("parse --to: %w", err)
		}
		mode = "backfill"
		result, err = job.Backfill(ctx, fromTime, toTime)
		if err != nil {
			return err
		}
	} else {
		result, err = job.Run(ctx)
		if err != nil {
			return err
		}
	}

	logger.Info("link analytics job completed",
		"mode", mode,
		"skipped", result.Skipped,
		"current_hour", result.CurrentHourProcessed,
		"previous_hour", result.PreviousHourProcessed,
		"hours_processed", result.HoursProcessed,
		"gap_hours", result.GapHours,
		"clicks", result.Clicks,
	)
	return nil
}
