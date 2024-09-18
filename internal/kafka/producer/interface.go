package producer

import "context"

type Producer interface {
	SendMessage(ctx context.Context, topic string, data interface{}) error
}
