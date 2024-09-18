package postgres

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
)

type repository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewRepository(ctx context.Context, logger *logrus.Logger) (*repository, error) {
	rawDB, err := sql.Open("postgres", os.Getenv("POSTGRES_DSN"))
	if err != nil {
		return nil, errors.Wrap(err, "sql.Open")
	}

	db := sqlx.NewDb(rawDB, "postgres")
	if err = db.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "db.PingContext")
	}

	return &repository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *repository) SaveTicket(ctx context.Context, ticket models.Ticket) error {
	query, args := sq.Insert("tickets").
		Columns("chat_id", "unit", "name", "description").
		Values(ticket.ChatID, ticket.Unit, ticket.Name, ticket.Description).
		PlaceholderFormat(sq.Dollar).
		MustSql()

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "ExecContext")
	}

	r.logger.Infof("Saved msg %v to DB", ticket)

	return nil
}

func (r *repository) SaveAdmins(ctx context.Context, chatIDs []int64, unit string) error {
	builder := sq.Insert("admins").Columns("chat_id", "unit")

	for _, v := range chatIDs {
		builder = builder.Values(v, unit)
	}

	query, args := builder.Suffix("on conflict do nothing").
		PlaceholderFormat(sq.Dollar).
		MustSql()

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "ExecContext")
	}

	r.logger.Infof("Saved admins %s to DB", unit)

	return nil
}

func (r *repository) GetAdmins(ctx context.Context, unit string) ([]models.Admin, error) {
	query, args := sq.Select("chat_id").
		From("admins").
		Where(sq.Eq{"unit": unit}).
		PlaceholderFormat(sq.Dollar).
		MustSql()

	var admins []models.Admin

	if err := r.db.SelectContext(ctx, &admins, query, args...); err != nil {
		return nil, errors.Wrap(err, "SelectContext")
	}

	return admins, nil
}
