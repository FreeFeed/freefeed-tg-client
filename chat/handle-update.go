package chat

import tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (c *Chat) HandleUpdate(update tg.Update) {
	c.handleMessage(update)
	c.handleCallback(update)
	c.handleCommand(update)
	c.printExpectationMessage()
}
