package message_processor

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goddeuce1/tg_bot_tz/internal/kafka/producer"
	"github.com/goddeuce1/tg_bot_tz/internal/repository"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/redis"
	"github.com/goddeuce1/tg_bot_tz/internal/tickets/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strings"
)

type messageProcessor struct {
	repo          repository.Repository
	stateOperator redis.StateOperator
	kafkaProducer producer.Producer
	logger        *logrus.Logger
	adminKey      string
}

func NewMessageProcessor(
	repo repository.Repository,
	redisDB redis.StateOperator,
	kafkaProducer producer.Producer,
	logger *logrus.Logger,
	adminKey string,
) *messageProcessor {
	return &messageProcessor{
		repo:          repo,
		stateOperator: redisDB,
		kafkaProducer: kafkaProducer,
		logger:        logger,
		adminKey:      adminKey,
	}
}

func (mp *messageProcessor) Process(ctx context.Context, processorData interface{}, botAPI *tgbotapi.BotAPI) error {
	msg, ok := processorData.(*tgbotapi.Message)
	if !ok {
		return errors.New("not message")
	}

	switch msg.Text {
	case commandStart:
		if err := mp.stateOperator.SetState(ctx, msg.Chat.ID, redis.ChatData{}); err != nil {
			return errors.Wrap(err, "stateOperator.SetState")
		}

		data := redis.ChatData{
			State: commandStart,
		}

		cmd := getCommand(data, msg.Text)

		msgToSend := tgbotapi.NewMessage(msg.Chat.ID, cmd.MessageText)
		if cmd.HasKeyboard {
			msgToSend.ReplyMarkup = cmd.KeyboardButton
		}

		if _, err := botAPI.Send(msgToSend); err != nil {
			return errors.Wrap(err, "botAPI.Send")
		}

	case commandNew:
		data := redis.ChatData{
			State: commandNew,
		}

		cmd := getCommand(data, msg.Text)

		data.State = commandUnit

		if err := mp.stateOperator.SetState(ctx, msg.Chat.ID, data); err != nil {
			return errors.Wrap(err, "stateOperator.SetState")
		}

		msgToSend := tgbotapi.NewMessage(msg.Chat.ID, cmd.MessageText)

		if cmd.HasKeyboard {
			msgToSend.ReplyMarkup = cmd.KeyboardButton
		}

		if _, err := botAPI.Send(msgToSend); err != nil {
			return errors.Wrap(err, "botAPI.Send")
		}

	case commandAdmin:
		data := redis.ChatData{
			State: commandAdmin,
		}

		cmd := getCommand(data, msg.Text)

		data.State = commandAdminKey

		if err := mp.stateOperator.SetState(ctx, msg.Chat.ID, data); err != nil {
			return errors.Wrap(err, "stateOperator.SetState")
		}

		msgToSend := tgbotapi.NewMessage(msg.Chat.ID, cmd.MessageText)
		msgToSend.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

		if _, err := botAPI.Send(msgToSend); err != nil {
			return errors.Wrap(err, "botAPI.Send")
		}

		// Обработка ввода пользователя
	default:
		state, err := mp.stateOperator.GetState(ctx, msg.Chat.ID)
		if err != nil {
			return errors.Wrap(err, "stateOperator.GetState")
		}

		// Состояние пользователя при опросе
		chatData := redis.ChatData{
			State: commandUnknown,
			Data:  state.Data,
		}

		switch state.State {
		case commandUnit:
			chatData.State = commandName
			chatData.Data.Unit = msg.Text

		case commandName:
			chatData.State = commandDescription
			chatData.Data.Name = msg.Text

		case commandDescription:
			chatData.State = commandDone
			chatData.Data.Description = msg.Text

		case commandAdminKey:
			return mp.processAdmin(ctx, msg, botAPI)

		case commandAdminLoggedIn:
			return nil

		default:
			chatData.State = commandUnknown
		}

		state.Data = chatData.Data

		// Отправка сообщения в чат
		textData := getCommand(*state, msg.Text)
		if textData.MessageText != "" {
			msgToSend := tgbotapi.NewMessage(msg.Chat.ID, textUnknown)

			msgToSend.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			msgToSend.Text = textData.MessageText

			if textData.HasKeyboard {
				msgToSend.ReplyMarkup = textData.KeyboardButton
			}

			if _, err = botAPI.Send(msgToSend); err != nil {
				return errors.Wrap(err, "botAPI.Send")
			}
		}

		// Состояние пользователя при опросе, сейвим в кеш с ттл
		if textData.MessageText != textUnknown && state.State != commandDone {
			if err = mp.stateOperator.SetState(ctx, msg.Chat.ID, chatData); err != nil {
				return errors.Wrap(err, "stateOperator.SetState")
			}
		}

		// Здесь логика с отправкой админу
		// Шлем тут в кафку (для контроля нагрузки)
		// Кафка в отдельном месте читает и записывает в постгрес
		// Затем шлет обращение в чат админа
		// Kafka UI на 8090 порту
		if msg.Text == textSubmitYes && state.State == commandDone {
			data := models.Ticket{
				ChatID:      msg.Chat.ID,
				Unit:        state.Data.Unit,
				Name:        state.Data.Name,
				Description: state.Data.Description,
			}

			if err = mp.kafkaProducer.SendMessage(ctx, "tickets", data); err != nil {
				return errors.Wrap(err, "kafkaProducer.SendMessage")
			}
		}
	}

	return nil
}

func (mp *messageProcessor) processAdmin(ctx context.Context, msg *tgbotapi.Message, botAPI *tgbotapi.BotAPI) error {
	data := redis.ChatData{
		State: commandAdminLoggedIn,
	}

	ss := strings.Split(msg.Text, " ")
	if _, exists := units[ss[0]]; !(exists && ss[1] == mp.adminKey) {
		data.State = commandAdminWrong
	}

	cmd := getCommand(data, msg.Text)

	if err := mp.stateOperator.SetState(ctx, msg.Chat.ID, data); err != nil {
		return errors.Wrap(err, "stateOperator.SetState")
	}

	msgToSend := tgbotapi.NewMessage(msg.Chat.ID, cmd.MessageText)

	if _, err := botAPI.Send(msgToSend); err != nil {
		return errors.Wrap(err, "botAPI.Send")
	}

	if err := mp.repo.SaveAdmins(ctx, []int64{msg.Chat.ID}, ss[0]); err != nil {
		return errors.Wrap(err, "repo.SaveAdmins")
	}

	return nil
}
