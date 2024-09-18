package message_processor

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goddeuce1/tg_bot_tz/internal/repository/redis"
)

const (
	commandStart = "/start"
	commandNew   = "/new"

	commandUnit        = "unit"
	commandName        = "name"
	commandDescription = "description"
	commandDone        = "done"
	commandUnknown     = "unknown"

	commandAdmin         = "/admin"
	commandAdminKey      = "admin_key"
	commandAdminLoggedIn = "admin_logged_in"
	commandAdminWrong    = "admin_wrong"

	textHello       = "Приветствую! Что вы бы вы хотели сделать?"
	textSelectUnit  = "Выберете подразделение"
	textName        = "Напишите заголовок обращения"
	textDescription = "Введите описание обращения"
	textSubmit      = `
Отлично! Ваша заявка:

Подразделение: %s
Название: %s
Описание: %s

Отправить администратору?`
	textSubmitYes     = "Отправить"
	textSubmitNo      = "Не отправлять"
	textSubmitSuccess = "Ваша заявка отправлена! Ожидайте, с вами скоро свяжутся"
	textSubmitFailure = "Если вы ошиблись при создании заявки, вы можете создать её снова"
	textUnknown       = "Я вас не понимаю :("

	textAdminKey      = "Введите подразделение (Поддержка, IT или Billing) и ключ через пробел"
	textAdminWrongKey = "Неправильно, в доступе отказано"
	textAdminWelcome  = "Добро пожаловать! Ожидайте обращений"

	unitSupport = "Поддержка"
	unitIT      = "IT"
	unitBilling = "Billing"
)

var units = map[string]bool{
	unitSupport: true,
	unitIT:      true,
	unitBilling: true,
}

type command struct {
	MessageText    string
	HasKeyboard    bool
	KeyboardButton tgbotapi.ReplyKeyboardMarkup
}

func getCommand(state redis.ChatData, text string) command {
	if state.State == "" {
		return command{}
	}

	switch state.State {
	case commandStart:
		if text != commandStart {
			return command{
				MessageText: textUnknown,
			}
		}

		return command{
			MessageText: textHello,
			HasKeyboard: true,
			KeyboardButton: tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(commandNew),
					tgbotapi.NewKeyboardButton(commandAdmin),
				),
			),
		}

	case commandNew:
		if text != commandNew {
			return command{
				MessageText: textUnknown,
			}
		}

		return command{
			MessageText: textSelectUnit,
			HasKeyboard: true,
			KeyboardButton: tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(unitSupport),
					tgbotapi.NewKeyboardButton(unitIT),
					tgbotapi.NewKeyboardButton(unitBilling),
				),
			),
		}

	case commandAdmin:
		if text != commandAdmin {
			return command{
				MessageText: textUnknown,
			}
		}

		return command{
			MessageText: textAdminKey,
		}

	case commandAdminLoggedIn:
		return command{
			MessageText: textAdminWelcome,
		}

	case commandAdminWrong:
		return command{
			MessageText: textAdminWrongKey,
		}

	case commandUnit:
		if _, exists := units[text]; !exists {
			return command{
				MessageText: textUnknown,
			}
		}

		return command{
			MessageText: textName,
		}

	case commandName:
		return command{
			MessageText: textDescription,
		}

	case commandDescription:
		data := state.Data

		return command{
			MessageText: fmt.Sprintf(textSubmit, data.Unit, data.Name, data.Description),
			HasKeyboard: true,
			KeyboardButton: tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(textSubmitYes),
					tgbotapi.NewKeyboardButton(textSubmitNo),
				),
			),
		}

	case commandDone:
		switch text {
		case textSubmitYes:
			return command{
				MessageText: textSubmitSuccess,
			}

		case textSubmitNo:
			return command{
				MessageText: textSubmitFailure,
				HasKeyboard: true,
				KeyboardButton: tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton(commandNew),
					),
				),
			}

		default:
			return command{
				MessageText: textUnknown,
			}
		}

	default:
		return command{
			MessageText: textUnknown,
		}
	}
}
