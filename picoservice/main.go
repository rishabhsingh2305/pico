package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"picoservice/internal/app"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/lambdahttp"
)

const closeTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

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
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if err := db.Close(ctx); err != nil {
			logger.Error("close mongodb", "error", err)
		}
	}()

	handler, err := app.New(logger, db, cfg)
	if err != nil {
		logger.Error("build application", "error", err)
		os.Exit(1)
	}

	adapter := lambdahttp.New(handler, logger)
	lambda.Start(adapter.Handle)
}
