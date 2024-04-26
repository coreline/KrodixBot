package internal

import (
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type telegram struct {
	bot *telego.Bot
	log *logger
}

func newBot(bot *telego.Bot, log *logger) *telegram {
	return &telegram{
		bot: bot,
		log: log,
	}
}

func (me *telegram) newMessageChat(chatId int64, format string, a ...any) *telego.SendMessageParams {
	return tu.Message(tu.ID(chatId), fmt.Sprintf(format, a...))
}

func (me *telegram) newMessage(message telego.MaybeInaccessibleMessage, format string, a ...any) *telego.SendMessageParams {
	return me.newMessageChat(message.GetChat().ID, format, a...)
}

func (me *telegram) sendMessage(message *telego.SendMessageParams) (msg *telego.Message, err error) {
	msg, err = me.bot.SendMessage(message)
	if err != nil {
		me.log.Errorf("SendMessage: %w", err)
	}
	return msg, err
}

func (me *telegram) deleteMessage(message telego.MaybeInaccessibleMessage) error {
	return me.bot.DeleteMessage(&telego.DeleteMessageParams{
		ChatID:    tu.ID(message.GetChat().ID),
		MessageID: message.GetMessageID(),
	})
}

func (me *telegram) changeMessage(old telego.MaybeInaccessibleMessage, new *telego.SendMessageParams) (msg *telego.Message, err error) {
	replyMarkup, ok := new.ReplyMarkup.(*telego.InlineKeyboardMarkup)
	if ok {
		msg, err = me.bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:             tu.ID(old.GetChat().ID),
			MessageID:          old.GetMessageID(),
			Text:               new.Text,
			ParseMode:          new.ParseMode,
			Entities:           new.Entities,
			LinkPreviewOptions: new.LinkPreviewOptions,
			ReplyMarkup:        replyMarkup,
		})
		if err != nil {
			me.deleteMessage(old)
			msg, err = me.sendMessage(new)
		}
	} else {
		me.deleteMessage(old)
		msg, err = me.sendMessage(new)
	}
	return msg, err
}

func (me *telegram) changeMessageText(old telego.MaybeInaccessibleMessage, text string) (msg *telego.Message, err error) {
	return me.changeMessage(old, &telego.SendMessageParams{
		ChatID: tu.ID(old.GetChat().ID),
		Text:   text,
	})
}
