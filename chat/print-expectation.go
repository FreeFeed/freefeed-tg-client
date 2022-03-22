package chat

import (
	"github.com/FreeFeed/freefeed-tg-client/store"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/enescakir/emoji"
	"golang.org/x/text/message"
)

func (c *Chat) printExpectationMessage() {
	p := message.NewPrinter(c.State.Language)

	if c.State.Expectation == store.ExpectAuthToken {
		createTokenURL := "https://" + c.App.FreeFeedAPI().HostName +
			"/settings/app-tokens/create?title=FreeFeed%20Telegram%20bot&scopes=read-my-info%20read-realtime%20manage-notifications%20manage-posts%20manage-subscription-requests%20manage-groups"

		msg := c.newHTMLMessage(p.Sprintf("Please create the access token and send it to the bot:"))
		msg.ReplyMarkup = tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{
			tg.NewInlineKeyboardButtonURL(emoji.Parse(
				p.Sprintf(":key: Create token")),
				createTokenURL,
			),
		})
		c.ShouldSend(msg)

	} else if c.State.Expectation == store.ExpectLanguage {
		msg := c.newHTMLMessage("Hello :wave:\nBefore we continue, please choose your language:")
		msg.ReplyMarkup = tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{
			tg.NewInlineKeyboardButtonData(emoji.Parse(":flag-us: English"), "lang:en"),
			tg.NewInlineKeyboardButtonData(emoji.Parse(":flag-ru: Russian"), "lang:ru"),
		})
		c.ShouldSend(msg)

	} else if c.State.Expectation == store.ExpectComment {
		text := p.Sprintf("Enter your comment text.")
		if c.State.CommentPrefix != "" {
			text = p.Sprintf("Enter your comment text. The comment will be prefixed with \"%s\"", c.State.CommentPrefix)
		}

		msg := c.newHTMLMessage(text)
		msg.ReplyMarkup = tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{
			tg.NewInlineKeyboardButtonData(
				emoji.Parse(p.Sprintf(":no_entry_sign: Cancel")),
				"cancel",
			),
		})
		c.ShouldSend(msg)
	}
}
