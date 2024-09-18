package bot

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goddeuce1/tg_bot_tz/internal/bot/processor"
	"github.com/goddeuce1/tg_bot_tz/internal/bot/processor/callback_processor"
	"github.com/goddeuce1/tg_bot_tz/internal/bot/processor/message_processor"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/producer"
	"github.com/goddeuce1/tg_bot_tz/internal/repository"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/redis"
	"github.com/goddeuce1/tg_bot_tz/internal/workerpool"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

type Bot struct {
	botAPI            *tgbotapi.BotAPI
	messageProcessor  processor.Processor
	callbackProcessor processor.Processor
	workerPool        workerpool.JobRunner
	logger            *logrus.Logger
}

func NewBot(
	botAPI *tgbotapi.BotAPI,
	repo repository.Repository,
	redisDB redis.StateOperator,
	workerPool workerpool.JobRunner,
	kafkaProducer producer.Producer,
	logger *logrus.Logger,
) (*Bot, error) {
	return &Bot{
		botAPI:            botAPI,
		messageProcessor:  message_processor.NewMessageProcessor(repo, redisDB, kafkaProducer, logger, os.Getenv("ADMIN_KEY")),
		callbackProcessor: callback_processor.NewCallbackProcessor(logger),
		workerPool:        workerPool,
		logger:            logger,
	}, nil
}

func (b *Bot) Start(ctx context.Context, gracefulWg *sync.WaitGroup) {
	defer func() {
		b.logger.Info("Shutting down bot")

		gracefulWg.Done()
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.botAPI.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			msg := update.Message
			if msg != nil {
				b.workerPool.AddJob(func(ctx context.Context) error {
					err := b.messageProcessor.Process(ctx, msg, b.botAPI)

					if err != nil {
						b.logger.Errorf("Cannot process message ID = %d, Text = %s, Error = %s", msg.MessageID, msg.Text, err)
					} else {
						b.logger.Infof("Processed message ID = %d, Text = %s", msg.MessageID, msg.Text)
					}

					return nil
				})
			} else if update.CallbackQuery != nil {
				b.workerPool.AddJob(func(ctx context.Context) error {
					err := b.callbackProcessor.Process(ctx, update.CallbackQuery, b.botAPI)

					if err != nil {
						b.logger.Errorf("Cannot process callback ID = %s, Error = %s", update.CallbackQuery.ID, err)
					} else {
						b.logger.Infof("Processed callback ID = %s", update.CallbackQuery.ID)
					}

					return nil
				})
			}

		case <-ctx.Done():
			return
		}
	}
}
