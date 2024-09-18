package application

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goddeuce1/tg_bot_tz/internal/bot"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/models"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/producer"
	"github.com/goddeuce1/tg_bot_tz/internal/repository"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/postgres"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/redis"
	"github.com/goddeuce1/tg_bot_tz/internal/tickets"
	"github.com/goddeuce1/tg_bot_tz/internal/workerpool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"sync"
)

func Run(ctx context.Context, logger *logrus.Logger) error {
	// Postgres
	db, err := postgres.NewRepository(ctx, logger)
	if err != nil {
		return errors.Wrap(err, "postgres.NewRepository")
	}

	// Redis
	rdb, err := redis.NewRepository(ctx, logger)
	if err != nil {
		return errors.Wrap(err, "redis.NewRepository")
	}

	// Repo (db + cache)
	repo := repository.NewRepository(db, rdb, logger)

	// Workerpool
	workersCount, err := strconv.Atoi(os.Getenv("WORKERS_COUNT"))
	if err != nil {
		return errors.Wrap(err, "strconv.Atoi")
	}

	// For graceful shutdown
	wg := &sync.WaitGroup{}

	wg.Add(1)
	botPool := workerpool.NewWorkerPool(logger, uint32(workersCount), make(chan workerpool.Job))
	go botPool.Run(ctx, wg)

	// TG BotAPI
	token := os.Getenv("BOT_TOKEN")

	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return errors.Wrap(err, "tgbotapi.NewBotAPI")
	}

	// Kafka
	kafkaData := models.KafkaData{
		Brokers: []string{os.Getenv("KAFKA_BROKER")},
		ConsumerData: &models.ConsumerData{
			Group: "test_group",
			TopicsWithHandlers: map[string]models.HandlerFunc{
				"tickets": tickets.NewHandler(repo, botAPI, logger).Handle,
			},
		},
	}

	kafkaClient, kafkaErr := kafka.NewKafkaClient(ctx, logger, kafkaData, true)
	if kafkaErr != nil {
		return errors.Wrap(err, "NewKafkaClient: %w")
	}

	defer kafkaClient.Close()

	kafkaProducer := producer.NewProducer(kafkaClient.KafkaClient)

	// Tickets consumer
	consumer := kafka.NewConsumer(kafkaClient, logger)

	wg.Add(1)
	go consumer.Start(ctx, wg)

	tgBot, err := bot.NewBot(botAPI, repo, rdb, botPool, kafkaProducer, logger)
	if err != nil {
		return errors.Wrap(err, "bot.NewBot")
	}

	wg.Add(1)
	go tgBot.Start(ctx, wg)

	logger.Info("Application started")

	// Wait for graceful shutdown
	wg.Wait()
	logger.Info("Graceful shutdown")

	return nil
}
