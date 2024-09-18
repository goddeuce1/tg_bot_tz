package kafka

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/consts"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Client struct {
	KafkaClient  *kgo.Client
	splitConsume *SplitConsume
}

func NewKafkaClient(
	ctx context.Context,
	logger *logrus.Logger,
	kafkaData models.KafkaData,
	addDefaultOptions bool,
) (*Client, error) {
	if err := validate(kafkaData.ConsumerData, logger); err != nil {
		return nil, errors.Wrap(err, "validate")
	}

	client := &Client{}

	options := []kgo.Opt{
		kgo.SeedBrokers(kafkaData.Brokers...),
	}

	// producer options
	if addDefaultOptions {
		options = append(options, getProducerOptions()...)
	}

	// consumer options
	if kafkaData.ConsumerData != nil {
		consumerOptions := getConsumerOptions(kafkaData.ConsumerData, logger, client)
		if addDefaultOptions {
			options = append(options, consumerOptions...)
		}
	}

	// options override
	if len(kafkaData.Options) > 0 {
		options = append(options, kafkaData.Options...)
	}

	kafkaClient, err := kgo.NewClient(options...)
	if err != nil {
		return nil, errors.Wrap(err, "NewClient: %w")
	}

	if err = kafkaClient.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "Ping: %w")
	}

	client.KafkaClient = kafkaClient

	return client, nil
}

func (c *Client) Close() {
	c.KafkaClient.Close()
}

func validate(consumerData *models.ConsumerData, logger *logrus.Logger) error {
	if consumerData != nil {
		for topic, handler := range consumerData.TopicsWithHandlers {
			if handler == nil {
				return errors.Errorf("handler for topic %s must not be nil", topic)
			}
		}
	}

	if logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func getProducerOptions() []kgo.Opt {
	return []kgo.Opt{
		kgo.MetadataMaxAge(consts.MetadataMaxAge),
		kgo.MaxBufferedRecords(consts.MaxBufferedRecords),
		kgo.ProducerBatchMaxBytes(consts.ProducerBatchMaxBytes),
		kgo.RecordPartitioner(kgo.RoundRobinPartitioner()),
	}
}

func getConsumerOptions(consumerData *models.ConsumerData, logger *logrus.Logger, client *Client) []kgo.Opt {
	topics := make([]string, 0)
	for k := range consumerData.TopicsWithHandlers {
		topics = append(topics, k)
	}

	splitConsume := &SplitConsume{
		consumers:     make(map[models.TopicWithPartition]*partitionConsumer),
		logger:        logger,
		topicHandlers: consumerData.TopicsWithHandlers,
	}

	client.splitConsume = splitConsume

	return []kgo.Opt{
		kgo.ConsumerGroup(consumerData.Group),
		kgo.ConsumeTopics(topics...),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
		kgo.OnPartitionsAssigned(splitConsume.assigned),
		kgo.OnPartitionsRevoked(splitConsume.lost),
		kgo.OnPartitionsLost(splitConsume.lost),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
		kgo.FetchMaxBytes(consts.FetchMaxBytes),
		kgo.FetchMaxPartitionBytes(consts.FetchMaxPartitionBytes),
		kgo.ConsumePreferringLagFn(kgo.PreferLagAt(consts.PreferLagAt)),
		kgo.Balancers(kgo.RoundRobinBalancer()),
	}
}
