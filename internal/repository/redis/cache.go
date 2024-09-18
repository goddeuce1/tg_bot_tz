package redis

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/models"
	"github.com/pkg/errors"
	"strconv"
)

func (r *repository) SaveTicket(_ context.Context, _ models.Ticket) error {
	return nil
}

func (r *repository) SaveAdmins(ctx context.Context, chatIDs []int64, unit string) error {
	pipe := r.db.TxPipeline()

	for _, v := range chatIDs {
		pipe.LPush(ctx, unit, v)
	}

	pipe.Expire(ctx, unit, r.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(err, "Exec")
	}

	return nil
}

func (r *repository) GetAdmins(ctx context.Context, unit string) ([]models.Admin, error) {
	admins, err := r.db.LRange(ctx, unit, 0, -1).Result()
	if err != nil {
		return nil, errors.Wrap(err, "LRange")
	}

	response := make([]models.Admin, 0)
	for _, v := range admins {
		chatID, errConvert := strconv.Atoi(v)
		if errConvert != nil {
			return nil, errors.Wrap(errConvert, "strconv.Atoi")
		}

		admin := models.Admin{
			ChatID: int64(chatID),
			Unit:   unit,
		}

		response = append(response, admin)
	}

	return response, nil
}
