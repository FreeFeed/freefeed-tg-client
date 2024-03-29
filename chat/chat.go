package chat

import (
	"fmt"

	"github.com/FreeFeed/freefeed-tg-client/frf"
	"github.com/FreeFeed/freefeed-tg-client/store"
	"github.com/davidmz/debug-log"
	"github.com/enescakir/emoji"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Chat struct {
	ID    ID
	State *store.State
	App   App

	dLog debug.Logger
	eLog debug.Logger
}

func New(id ID, app App) (*Chat, error) {
	state, err := app.LoadState(id)
	if err != nil {
		return nil, err
	}
	return &Chat{
		ID:    id,
		App:   app,
		State: state,
		dLog:  app.DebugLog().Fork(app.DebugLog().Name() + fmt.Sprintf(":chat:%d", id)),
		eLog:  app.ErrorLog().Fork(app.ErrorLog().Name() + fmt.Sprintf(":chat:%d", id)),
	}, nil
}

func (c *Chat) debugLog() debug.Logger { return c.dLog }
func (c *Chat) errorLog() debug.Logger { return c.eLog }

func (c *Chat) frfAPIWithToken(accessToken string) *frf.API {
	api := c.App.FreeFeedAPI()
	api.AccessToken = accessToken
	return api
}

func (c *Chat) frfAPI() *frf.API { return c.frfAPIWithToken(c.State.AccessToken) }

func (c *Chat) saveState() error   { return c.ShouldOK(c.App.SaveState(c.State)) }
func (c *Chat) deleteState() error { return c.ShouldOK(c.App.DeleteState(c.ID)) }

func (c *Chat) ShouldSend(msg tg.Chattable) (tg.Message, error) {
	m, err := c.Should(c.App.Send(msg))
	return m.(tg.Message), err
}

func (c *Chat) ShouldSendAndSave(msg tg.Chattable, rec store.SentMsgRec) (tg.Message, error) {
	m, err := c.Should(c.App.Send(msg))
	if err == nil {
		rec.MessageID = m.(tg.Message).MessageID
		c.ShouldOK(c.App.PutMsgRec(c.ID, rec))
	}
	return m.(tg.Message), err
}

func (c *Chat) newHTMLMessage(text string) *tg.MessageConfig {
	return c.newRawHTMLMessage(c.App.Linkify(emoji.Parse(text)))
}

func (c *Chat) newRawHTMLMessage(html string) *tg.MessageConfig {
	msg := tg.NewMessage(c.ID, html)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	return &msg
}
