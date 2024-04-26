package internal

import (
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type Client struct {
	tg     *telegram
	log    *logger
	anchat *anchat
}

func NewClient(token string, debugMode bool, printErrors bool) (client Client, err error) {
	log := newDefaultLogger(debugMode, printErrors)

	bot, err := telego.NewBot(token, telego.WithDefaultLogger(debugMode, printErrors))
	if err != nil {
		return
	}

	tg := newBot(bot, log)

	anchat := newAnchat(tg, log)

	client = Client{
		tg:     tg,
		log:    log,
		anchat: anchat,
	}
	return client, nil
}

func (me *Client) Run() error {
	updates, err := me.tg.bot.UpdatesViaLongPolling(nil)
	if err != nil {
		return err
	}
	defer me.tg.bot.StopLongPolling()

	bh, err := th.NewBotHandler(me.tg.bot, updates)
	if err != nil {
		return err
	}
	defer bh.Stop()

	// Команда
	handleCommand := func(command string, handle func(telego.MaybeInaccessibleMessage) (string, error)) {
		bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
			handle(&message)
		}, th.CommandEqual(command))
	}
	handleCommand("start", me.anchat.menu)
	handleCommand("search", me.anchat.menu)
	handleCommand("next", me.anchat.next)
	handleCommand("stop", me.anchat.stop)
	handleCommand("help", me.anchat.help)

	// Сообщение получено
	bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
		if me.anchat.hasOpponent(&message) {
			me.anchat.sendMessage(&message)
		}
	}, th.AnyMessage())

	// Callback
	bh.HandleCallbackQuery(func(bot *telego.Bot, query telego.CallbackQuery) {
		args := strings.Split(query.Data, "-")
		text := ""
		if len(args) == 0 {
			return
		}
		if args[0] == "anchat" {
			text = me.anchat.callbackQuery(args[1:], &query)
		}
		if text == "" {
			_ = bot.AnswerCallbackQuery(tu.CallbackQuery(query.ID))
		} else {
			_ = bot.AnswerCallbackQuery(tu.CallbackQuery(query.ID).WithText(text))
		}
	}, th.AnyCallbackQueryWithMessage(), th.AnyCallbackQuery())

	bh.Start()
	return nil
}
