package main

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/application"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)
	defer stop()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	if err := godotenv.Load(); err != nil {
		logger.Fatalf("godotenv.Load: %s", err)
	}

	if err := application.Run(ctx, logger); err != nil {
		logger.Fatalf("application.Run: %s", err)
	}
}
