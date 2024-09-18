package kafka

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/models"
	"github.com/sirupsen/logrus"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
)

type partitionConsumer struct {
	kafkaClient *kgo.Client
	topic       string
	partition   int32
	handler     models.HandlerFunc
	logger      *logrus.Logger

	quit chan struct{}
	done chan struct{}
	recs chan []*kgo.Record
}

func (pc *partitionConsumer) consume(ctx context.Context) {
	defer close(pc.done)

	for {
		select {
		case <-pc.quit:
			return

		case records := <-pc.recs:
			successRecords := make([]*kgo.Record, 0)

			for _, v := range records {
				if err := pc.handler(ctx, v.Value); err != nil {
					pc.logger.Error(
						"error during handler work in consumer",
						pc.topic, pc.partition, err,
					)
				} else {
					successRecords = append(successRecords, v)
				}
			}

			if err := pc.kafkaClient.CommitRecords(ctx, successRecords...); err != nil {
				pc.logger.Error(
					"error during committing records",
					pc.topic, pc.partition, err,
				)
			}
		}
	}
}

type Consumers map[models.TopicWithPartition]*partitionConsumer

type SplitConsume struct {
	consumers     Consumers
	logger        *logrus.Logger
	topicHandlers map[string]models.HandlerFunc
}

func (s *SplitConsume) assigned(ctx context.Context, kafkaClient *kgo.Client, assigned map[string][]int32) {
	for topic, partitions := range assigned {
		handler, ok := s.topicHandlers[topic]
		if !ok {
			continue
		}

		for _, partition := range partitions {
			partitionConsumer := &partitionConsumer{
				kafkaClient: kafkaClient,
				topic:       topic,
				partition:   partition,
				handler:     handler,
				logger:      s.logger,

				quit: make(chan struct{}),
				done: make(chan struct{}),
				recs: make(chan []*kgo.Record, 5), //nolint
			}

			topicWithPartition := models.TopicWithPartition{
				Topic:     topic,
				Partition: partition,
			}

			s.consumers[topicWithPartition] = partitionConsumer

			go partitionConsumer.consume(ctx)
		}
	}
}

func (s *SplitConsume) lost(_ context.Context, _ *kgo.Client, lost map[string][]int32) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for topic, partitions := range lost {
		for _, partition := range partitions {
			topicWithPartition := models.TopicWithPartition{
				Topic:     topic,
				Partition: partition,
			}

			partitionConsumer := s.consumers[topicWithPartition]

			delete(s.consumers, topicWithPartition)
			close(partitionConsumer.quit)

			s.logger.Info("waiting for work to finish ", topic, partition)

			wg.Add(1)

			go func() {
				defer wg.Done()

				<-partitionConsumer.done
			}()
		}
	}
}
