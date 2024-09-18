package tickets

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goddeuce1/tg_bot_tz/internal/repository"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/models"
	hmodels "github.com/goddeuce1/tg_bot_tz/internal/tickets/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

type handler struct {
	repo   repository.Repository
	botAPI *tgbotapi.BotAPI
	logger *logrus.Logger
}

func NewHandler(
	repo repository.Repository,
	botAPI *tgbotapi.BotAPI,
	logger *logrus.Logger,
) *handler {
	return &handler{
		repo:   repo,
		botAPI: botAPI,
		logger: logger,
	}
}

func (h *handler) Handle(ctx context.Context, data []byte) error {
	var ticket hmodels.Ticket

	if err := jsoniter.Unmarshal(data, &ticket); err != nil {
		return errors.Wrap(err, "jsoniter.Unmarshal")
	}

	dbTicket := models.Ticket{
		ChatID:      ticket.ChatID,
		Unit:        ticket.Unit,
		Name:        ticket.Name,
		Description: ticket.Description,
	}

	if err := h.repo.SaveTicket(ctx, dbTicket); err != nil {
		return errors.Wrap(err, "repo.Save")
	}

	admins, err := h.repo.GetAdmins(ctx, ticket.Unit)
	if err != nil {
		return errors.Wrap(err, "repo.GetAdmins")
	}

	adminsMap := make(map[int64]bool)
	for _, v := range admins {
		adminsMap[v.ChatID] = true
	}

	for k, _ := range adminsMap {
		text, markup := buildAdminMessageWithMarkup(ticket)

		msgToSend := tgbotapi.NewMessage(k, text)
		msgToSend.ReplyMarkup = markup

		if _, err = h.botAPI.Send(msgToSend); err != nil {
			return errors.Wrap(err, "botAPI.Send")
		}
	}

	return nil
}

func buildAdminMessageWithMarkup(ticket hmodels.Ticket) (string, tgbotapi.InlineKeyboardMarkup) {
	msg := `
Пришло новое обращение:

Название - %s
Описание - %s

Необходимо ответить клиенту #%d`

	msg = fmt.Sprintf(msg, ticket.Name, ticket.Description, ticket.ChatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Fast Reply", "Hello, world!"),
		),
	)

	return msg, keyboard
}
