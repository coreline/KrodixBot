package internal

// TODO:
// 1. в newAnChat считать всех юзеров
// 2. confirm_sex - сохранение пола в БД
import (
	"fmt"
	"strconv"
	"sync"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

const max_queue_length = 100
const (
	sex_unknown = 0
	sex_male    = 1
	sex_female  = 2
	sex_any     = 3
)

type dialog struct {
	sex      int
	seek     int
	seekId   int
	chatId   int64
	tagsId   [4]int
	messages map[int]int
}

type queue struct {
	seekId int
	chatId int64
}

type anchat struct {
	tg      *telegram
	log     *logger
	dialogs map[int64]*dialog
	queue   []queue
	mutex   sync.Mutex
}

func newAnchat(tg *telegram, log *logger) *anchat {
	return &anchat{
		tg:      tg,
		log:     log,
		dialogs: make(map[int64]*dialog),
		queue:   []queue{},
	}
}

func (me *anchat) callbackQuery(args []string, query *telego.CallbackQuery) string {
	var err error
	var status string
	cmd := "menu"
	arg := ""

	switch len(args) {
	case 1:
		cmd = args[0]
	case 2:
		cmd = args[0]
		arg = args[1]
	}

	switch cmd {
	case "menu":
		status, err = me.menu(query.Message)
	case "confirm_sex":
		status, err = me.confirmSex(arg, query.Message)
	case "set_sex":
		status, err = me.setSex(arg, query.Message)
	case "next":
		status, err = me.next(query.Message)
	case "find":
		status, err = me.find(arg, query.Message)
	case "like":
	case "dislike":
	case "ban":
	case "dummy":
		me.tg.deleteMessage(query.Message)
	}

	if err != nil {
		return err.Error()
	}
	return status
}

func (me *anchat) getDialogId(chatId int64) (result *dialog) {
	result, ok := me.dialogs[chatId]
	if !ok {
		result = &dialog{
			messages: make(map[int]int),
		}
		me.dialogs[chatId] = result
	}
	return result
}

func (me *anchat) getDialog(message telego.MaybeInaccessibleMessage) (result *dialog) {
	return me.getDialogId(message.GetChat().ID)
}

func (me *anchat) menu(message telego.MaybeInaccessibleMessage) (status string, err error) {
	dialog := me.getDialog(message)
	if dialog.sex == sex_unknown {
		reply := me.tg.newMessage(message, "Для поиска собеседника укажите свой пол.")
		reply.WithReplyMarkup(tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("👱🏻‍♂️ Я парень").WithCallbackData(fmt.Sprintf("anchat-confirm_sex-%d", sex_male)),
				tu.InlineKeyboardButton("👩‍🦳 Я девушка").WithCallbackData(fmt.Sprintf("anchat-confirm_sex-%d", sex_female)),
			),
		))
		_, err = me.tg.sendMessage(reply)
	} else if dialog.chatId == 0 {
		reply := me.tg.newMessage(message, "С кем хотите поговорить?")
		reply.WithReplyMarkup(tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("🎲 Не важно").WithCallbackData(fmt.Sprintf("anchat-find-%d", sex_any)),
				tu.InlineKeyboardButton("👱🏻‍♂️ С парнем").WithCallbackData(fmt.Sprintf("anchat-find-%d", sex_male)),
				tu.InlineKeyboardButton("👩‍🦳 С девушкой").WithCallbackData(fmt.Sprintf("anchat-find-%d", sex_female)),
			),
		))
		_, err = me.tg.sendMessage(reply)
	} else {
		reply := me.tg.newMessage(message, "Завершить текущий диалог?")
		reply.WithReplyMarkup(tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("Да").WithCallbackData("anchat-next"),
				tu.InlineKeyboardButton("Нет").WithCallbackData("anchat-dummy"),
			),
		))
		_, err = me.tg.sendMessage(reply)
	}
	return status, err
}

func (me *anchat) confirmSex(sex string, message telego.MaybeInaccessibleMessage) (status string, err error) {
	reply := me.tg.newMessage(message, "Подтвердите ваш пол. Изменить решение будет невозможно.")
	reply.WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✅ Подтверждаю").WithCallbackData(fmt.Sprintf("anchat-set_sex-%s", sex)),
			tu.InlineKeyboardButton("❌ Отмена").WithCallbackData("anchat-menu"),
		),
	))
	_, err = me.tg.changeMessage(message, reply)
	if err != nil {
		err = fmt.Errorf("возникла ошибка, попробуйте повторить действие")
	}
	return status, err
}

func (me *anchat) setSex(sex string, message telego.MaybeInaccessibleMessage) (status string, err error) {
	dialogSelf := me.getDialog(message)
	value, err := strconv.Atoi(sex)
	if err != nil {
		err = fmt.Errorf("ошибка установки пола")
	} else {
		status = "Успешно подтверждено"
		dialogSelf.sex = value
		me.menu(message)
	}
	return status, err
}

func (me *anchat) next(message telego.MaybeInaccessibleMessage) (status string, err error) {
	status, err = me.stop(message)
	if err != nil {
		return status, err
	}
	return me.findInQueue(message)
}

func (me *anchat) help(message telego.MaybeInaccessibleMessage) (status string, err error) {
	return status, err
}

func (me *anchat) stop(message telego.MaybeInaccessibleMessage) (status string, err error) {
	dialogSelf := me.getDialog(message)
	if dialogSelf.chatId == 0 {
		return status, err
	}
	dialogSide := me.getDialogId(dialogSelf.chatId)

	if dialogSide.chatId != 0 {
		go func(chatId int64, oppId int64) {
			reply := me.tg.newMessageChat(chatId, "Собеседник закончил диалог")
			reply.WithReplyMarkup(tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("Начать новый диалог").WithCallbackData("anchat-next"),
				),
			))
			me.tg.sendMessage(reply)
		}(dialogSelf.chatId, message.GetChat().ID)
	}

	me.tg.sendMessage(me.tg.newMessage(message, "Диалог завершен"))

	dialogSelf.chatId = 0
	dialogSide.chatId = 0

	return status, err
}

func (me *anchat) find(sex string, message telego.MaybeInaccessibleMessage) (status string, err error) {
	seek, err := strconv.Atoi(sex)
	if err != nil || !(seek == sex_male || seek == sex_female || seek == sex_any) {
		return status, fmt.Errorf("ошибка поиска")
	}

	dialogSelf := me.getDialog(message)
	dialogSelf.seek = seek
	dialogSelf.seekId = 10*dialogSelf.sex + dialogSelf.seek

	switch dialogSelf.sex {
	case sex_male:
		switch dialogSelf.seek {
		case sex_male:
			dialogSelf.tagsId[0] = 10*sex_male + sex_male
			dialogSelf.tagsId[1] = 10*sex_male + sex_any
			dialogSelf.tagsId[2] = 0
			dialogSelf.tagsId[3] = 0
		case sex_female:
			dialogSelf.tagsId[0] = 10*sex_female + sex_male
			dialogSelf.tagsId[1] = 10*sex_female + sex_any
			dialogSelf.tagsId[2] = 0
			dialogSelf.tagsId[3] = 0
		case sex_any:
			dialogSelf.tagsId[0] = 10*sex_male + sex_male
			dialogSelf.tagsId[1] = 10*sex_male + sex_any
			dialogSelf.tagsId[2] = 10*sex_female + sex_male
			dialogSelf.tagsId[3] = 10*sex_female + sex_any
		}
	case sex_female:
		switch dialogSelf.seek {
		case sex_male:
			dialogSelf.tagsId[0] = 10*sex_male + sex_female
			dialogSelf.tagsId[1] = 10*sex_male + sex_any
			dialogSelf.tagsId[2] = 0
			dialogSelf.tagsId[3] = 0
		case sex_female:
			dialogSelf.tagsId[0] = 10*sex_female + sex_female
			dialogSelf.tagsId[1] = 10*sex_female + sex_any
			dialogSelf.tagsId[2] = 0
			dialogSelf.tagsId[3] = 0
		case sex_any:
			dialogSelf.tagsId[0] = 10*sex_male + sex_female
			dialogSelf.tagsId[1] = 10*sex_male + sex_any
			dialogSelf.tagsId[2] = 10*sex_female + sex_female
			dialogSelf.tagsId[3] = 10*sex_female + sex_any
		}
	default:
		return me.menu(message)
	}

	return me.findInQueue(message)
}

func (me *anchat) hasOpponentId(chatId int64) bool {
	dialogSelf := me.getDialogId(chatId)
	return dialogSelf.chatId > 0
}

func (me *anchat) hasOpponent(message telego.MaybeInaccessibleMessage) bool {
	return me.hasOpponentId(message.GetChat().ID)
}

func (me *anchat) sendMessage(message *telego.Message) (err error) {
	dialogSelf := me.getDialog(message)
	dialogSide := me.getDialogId(dialogSelf.chatId)

	sended, err := me.tg.bot.SendMessage(&telego.SendMessageParams{
		ChatID: tu.ID(dialogSelf.chatId),
		Text:   message.Text,
	})

	if err == nil && sended != nil {
		dialogSelf.messages[message.MessageID] = sended.MessageID
		dialogSide.messages[sended.MessageID] = message.MessageID
	}

	return err
}

func (me *anchat) findInQueue(message telego.MaybeInaccessibleMessage) (status string, err error) {
	me.mutex.Lock()
	defer me.mutex.Unlock()

	chatId := message.GetChat().ID
	dialogSelf := me.getDialogId(chatId)

	go me.tg.deleteMessage(message)

	selfIndex := -1
	sideIndex := -1
	for i, queue := range me.queue {
		if queue.chatId == chatId {
			selfIndex = i
			break
		}
	}

	for i, queue := range me.queue {
		if i == selfIndex {
			continue
		}
		for _, seekId := range dialogSelf.tagsId {
			if queue.seekId == seekId {
				sideIndex = i
				break
			}
		}
		if sideIndex > -1 {
			break
		}
	}

	if sideIndex < 0 {
		if len(me.queue) >= max_queue_length {
			for i := range me.queue {
				if i != selfIndex {
					sideIndex = i
					break
				}
			}
		} else {
			if selfIndex < 0 {
				me.queue = append(me.queue, queue{
					chatId: chatId,
					seekId: dialogSelf.seekId,
				})
				status = "Ищем подходящего собеседника"
			} else {
				if me.queue[selfIndex].seekId != dialogSelf.seekId {
					me.queue[selfIndex].seekId = dialogSelf.seekId
					status = "Настройки поиска изменены"
				} else {
					status = "Ищем подходящего собеседника"
				}
			}
		}
	}
	if sideIndex > -1 {
		queue := me.queue[sideIndex]
		if sideIndex > selfIndex {
			me.removeQueue(sideIndex)
			me.removeQueue(selfIndex)
		} else {
			me.removeQueue(selfIndex)
			me.removeQueue(sideIndex)
		}
		return status, me.connect(chatId, queue.chatId)
	}

	return status, err
}

func (me *anchat) connect(self int64, side int64) error {
	dialogSelf := me.getDialogId(self)
	dialogSide := me.getDialogId(side)
	clear(dialogSelf.messages)
	clear(dialogSide.messages)
	dialogSelf.chatId = side
	dialogSide.chatId = self

	send := func(chatId int64) {
		me.tg.sendMessage(me.tg.newMessageChat(chatId,
			"Соединение установлено. Поздоровайтесь с собеседником.",
		))
	}

	go send(self)
	go send(side)

	return nil
}

func (me *anchat) removeQueue(i int) {
	if i > -1 && i < len(me.queue) {
		me.queue = append(me.queue[:i], me.queue[i+1:]...)
	}
}
