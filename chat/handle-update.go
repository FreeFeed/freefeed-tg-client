package chat

import tg "github.com/davidmz/telegram-bot-api"

func (c *Chat) HandleUpdate(update tg.Update) {
	c.handleMessage(update)
	c.handleCallback(update)
	c.handleCommand(update)
	c.printExpectationMessage()
}
