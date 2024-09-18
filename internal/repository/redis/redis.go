package redis

import (
	"context"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type repository struct {
	db     *redis.Client
	logger *logrus.Logger
	ttl    time.Duration
}

func NewRepository(ctx context.Context, logger *logrus.Logger) (*repository, error) {
	opts := &redis.Options{
		Addr:            os.Getenv("REDIS_HOST"),
		DialTimeout:     5 * time.Second,
		ReadTimeout:     100 * time.Millisecond,
		WriteTimeout:    100 * time.Millisecond,
		ConnMaxIdleTime: 5 * time.Second,
	}

	db := redis.NewClient(opts)
	if _, err := db.Ping(ctx).Result(); err != nil {
		return nil, errors.Wrap(err, "db.Ping")
	}

	return &repository{
		db:     db,
		logger: logger,
		ttl:    1 * time.Minute,
	}, nil
}
