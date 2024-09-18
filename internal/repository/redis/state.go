package redis

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"strconv"
)

type StateOperator interface {
	GetState(ctx context.Context, chatID int64) (*ChatData, error)
	SetState(ctx context.Context, chatID int64, data ChatData) error
}

type ChatData struct {
	State string `redis:"state"`
	Data  Data   `redis:"data"`
}

type Data struct {
	Unit        string `json:"unit"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (d Data) MarshalBinary() ([]byte, error) {
	return jsoniter.Marshal(d)
}

func (r *repository) GetState(ctx context.Context, chatID int64) (*ChatData, error) {
	key := strconv.Itoa(int(chatID))

	d, err := r.db.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, errors.Wrap(err, "HSet")
	}

	var data Data
	if d["data"] != "" {
		if err = jsoniter.UnmarshalFromString(d["data"], &data); err != nil {
			return nil, errors.Wrap(err, "UnmarshalFromString")
		}
	}

	return &ChatData{
		State: d["state"],
		Data:  data,
	}, nil
}

func (r *repository) SetState(ctx context.Context, chatID int64, data ChatData) error {
	key := strconv.Itoa(int(chatID))

	pipe := r.db.TxPipeline()

	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, r.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(err, "Exec")
	}

	return nil
}
