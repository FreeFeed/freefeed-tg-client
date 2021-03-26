package chat

import (
	"github.com/davidmz/freefeed-tg-client/store"
	tg "github.com/davidmz/telegram-bot-api"
	"golang.org/x/text/message"
)

func (c *Chat) handleCommand(update tg.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	command := msg.Command()
	p := message.NewPrinter(c.State.Language)

	if command == "start" {
		if c.State.IsAuthorized() {
			c.ShouldSend(c.newHTMLMessage(
				p.Sprintf("We already know each other. Use the /logout command if you want to delete all of your data or start over."),
			))
		} else if !c.State.Language.IsRoot() {
			c.State.Expectation = store.ExpectAuthToken
			c.ShouldOK(c.saveState())
		}
	} else if command == "logout" && c.State.IsAuthorized() {
		c.ShouldSend(c.newHTMLMessage(
			p.Sprintf("OK, we will remove all of your data now. Use the /start command if you want to come back."),
		))
		c.App.StopRealtime(c.ID)
		c.ShouldOK(c.deleteState())
	} else if command == "load" && c.State.IsAuthorized() {
		// Load notifications from the server

		events, err := c.frfAPI().GetEvents()
		if err != nil {
			c.ShouldSend(c.newHTMLMessage(p.Sprintf(":alien: Cannot load events: %v", err)))
			return
		}
		c.ProcessEvents(events)

	} else if command == "pause" && c.State.IsAuthorized() {
		c.App.PauseEvents(c.ID)
		c.ShouldSend(c.newHTMLMessage(
			p.Sprintf("Your updates are paused now."),
		))

	} else if command == "resume" && c.State.IsAuthorized() {
		c.App.ResumeEvents(c.ID)
		c.ShouldSend(c.newHTMLMessage(
			p.Sprintf("Your updates are resumed now."),
		))

	} else if command == "whoami" && c.State.IsAuthorized() {
		user, err := c.frfAPI().GetMe()
		if err != nil {
			c.ShouldSend(c.newHTMLMessage(
				p.Sprintf("Cannot load user information: %v", err),
			))
		} else {
			c.ShouldSend(c.newHTMLMessage(
				p.Sprintf("You are using this bot as %s. Use the /logout command "+
					"if you want to delete all of your data or start as another user.",
					user),
			))
		}

	} else if command != "" {
		c.ShouldSend(c.newHTMLMessage(p.Sprintf(":alien: Unknown command")))
	}
}
