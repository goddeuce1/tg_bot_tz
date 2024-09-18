package callback_processor

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type callbackProcessor struct {
	logger *logrus.Logger
}

func NewCallbackProcessor(logger *logrus.Logger) *callbackProcessor {
	return &callbackProcessor{
		logger: logger,
	}
}

func (cp *callbackProcessor) Process(ctx context.Context, processorData interface{}, botAPI *tgbotapi.BotAPI) error {
	callback, ok := processorData.(*tgbotapi.CallbackQuery)
	if !ok {
		return errors.New("not callback")
	}

	ss := strings.Split(callback.Message.Text, "#")
	if len(ss) != 2 {
		return errors.New("strings.Split failed in callback")
	}

	clientChatID, err := strconv.Atoi(ss[1])
	if err != nil {
		return errors.Wrap(err, "strconv.Atoi")
	}

	msgToSend := tgbotapi.NewMessage(int64(clientChatID), callback.Data)
	if _, err = botAPI.Send(msgToSend); err != nil {
		return errors.Wrap(err, "botAPI.Send")
	}

	return nil
}
