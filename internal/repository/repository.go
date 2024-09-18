package repository

import (
	"context"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Repository interface {
	SaveTicket(ctx context.Context, ticket models.Ticket) error
	SaveAdmins(ctx context.Context, chatIDs []int64, unit string) error

	GetAdmins(ctx context.Context, unit string) ([]models.Admin, error)
}

type repository struct {
	repo   Repository
	cache  Repository
	logger *logrus.Logger
}

func NewRepository(repo Repository, cache Repository, logger *logrus.Logger) *repository {
	return &repository{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (r *repository) SaveTicket(ctx context.Context, ticket models.Ticket) error {
	if err := r.repo.SaveTicket(ctx, ticket); err != nil {
		return errors.Wrap(err, "repo.SaveTicket")
	}

	return nil
}

func (r *repository) SaveAdmins(ctx context.Context, chatIDs []int64, unit string) error {
	// В первую очередь сейвим в постгрю
	if err := r.repo.SaveAdmins(ctx, chatIDs, unit); err != nil {
		return errors.Wrap(err, "repo.SaveAdmin")
	}

	if err := r.cache.SaveAdmins(ctx, chatIDs, unit); err != nil {
		return errors.Wrap(err, "cache.GetAdmins")
	}

	return nil
}

func (r *repository) GetAdmins(ctx context.Context, unit string) ([]models.Admin, error) {
	// Сначала ищем в кеше
	admins, err := r.cache.GetAdmins(ctx, unit)
	if err != nil || len(admins) == 0 {
		admins, err = r.repo.GetAdmins(ctx, unit)
		if err != nil {
			return nil, errors.Wrap(err, "repo.GetAdmins")
		}

		chatIDs := make([]int64, 0)
		for _, v := range admins {
			chatIDs = append(chatIDs, v.ChatID)
		}

		if err = r.cache.SaveAdmins(ctx, chatIDs, unit); err != nil {
			return nil, errors.Wrap(err, "cache.SaveAdmins")
		}
	}

	return admins, nil
}
