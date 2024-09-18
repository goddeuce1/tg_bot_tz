package models

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

type (
	HandlerFunc func(ctx context.Context, data []byte) error
)

type KafkaData struct {
	Brokers      []string
	ConsumerData *ConsumerData
	Options      []kgo.Opt // добавляет опции или оверрайдит существующие, если addDefaultOptions == true
}

type ConsumerData struct {
	Group              string
	TopicsWithHandlers map[string]HandlerFunc
}

type TopicWithPartition struct {
	Topic     string
	Partition int32
}
