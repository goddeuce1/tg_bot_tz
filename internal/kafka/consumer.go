package kafka

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/consts"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/models"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
)

type consumer struct {
	kafkaClient  *kgo.Client
	splitConsume *SplitConsume
	logger       *logrus.Logger
}

func NewConsumer(client *Client, logger *logrus.Logger) *consumer {
	return &consumer{
		kafkaClient:  client.KafkaClient,
		splitConsume: client.splitConsume,
		logger:       logger,
	}
}

func (c *consumer) Start(ctx context.Context, gracefulWg *sync.WaitGroup) {
	defer func() {
		c.logger.Info("Shutting down kafka consumer")

		gracefulWg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			c.logger.Warn(
				"closing consumer of topics",
				strings.Join(c.kafkaClient.GetConsumeTopics(), ","),
			)

			return

		default:
			fetches := c.kafkaClient.PollRecords(ctx, consts.MaxPollRecords)
			if fetches.IsClientClosed() {
				return
			}

			for _, err := range fetches.Errors() {
				c.logger.Error(
					"error during fetch",
					err.Topic, err.Partition, err.Err,
				)
			}

			fetches.EachPartition(func(p kgo.FetchTopicPartition) {
				topicWithPartition := models.TopicWithPartition{
					Topic:     p.Topic,
					Partition: p.Partition,
				}

				if consumer, ok := c.splitConsume.consumers[topicWithPartition]; ok && consumer != nil {
					c.splitConsume.consumers[topicWithPartition].recs <- p.Records
				}
			})

			c.kafkaClient.AllowRebalance()
		}
	}
}
