package producer

import (
	"context"
	"github.com/pkg/errors"

	jsoniter "github.com/json-iterator/go"
	"github.com/twmb/franz-go/pkg/kgo"
)

type producer struct {
	kafkaClient *kgo.Client
}

func NewProducer(kafkaClient *kgo.Client) *producer {
	return &producer{
		kafkaClient: kafkaClient,
	}
}

func (p *producer) SendMessage(ctx context.Context, topic string, data interface{}) error {
	value, err := jsoniter.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "Marshal")
	}

	record := &kgo.Record{
		Value: value,
		Topic: topic,
	}

	if err = p.kafkaClient.ProduceSync(ctx, record).FirstErr(); err != nil {
		return errors.Errorf(
			"failed to produce record %s to topic %s partition %d: %s",
			string(record.Value), record.Topic, record.Partition, err,
		)
	}

	return nil
}
