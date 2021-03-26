package chat

import (
	"github.com/davidmz/debug-log"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/store"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/enescakir/emoji"
)

type Chat struct {
	ID    ID
	State *store.State
	App   App
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
	}, nil
}

func (c *Chat) debugLog() debug.Logger { return c.App.DebugLog() }
func (c *Chat) errorLog() debug.Logger { return c.App.ErrorLog() }

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

func (c *Chat) ShouldAnswer(config tg.CallbackConfig) (tg.APIResponse, error) {
	resp, err := c.Should(c.App.Tg().AnswerCallbackQuery(config))
	return resp.(tg.APIResponse), err
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
