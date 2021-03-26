package chat

import (
	"errors"

	"github.com/davidmz/freefeed-tg-client/store"
	"github.com/davidmz/freefeed-tg-client/types"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/enescakir/emoji"
	"github.com/gofrs/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func (c *Chat) handleCallback(update tg.Update) {
	cbQuery := update.CallbackQuery
	if cbQuery == nil {
		return
	}

	msg := update.CallbackQuery.Message

	p := message.NewPrinter(c.State.Language)

	cbData := cbQuery.Data
	c.debugLog().Println("Callback Data: ", cbData)

	if cbData == "lang:ru" || cbData == "lang:en" {
		if cbData == "lang:ru" {
			c.State.Language = language.Russian
		} else {
			c.State.Language = language.English
		}
		c.ShouldOK(c.saveState())

		p := message.NewPrinter(c.State.Language)

		c.ShouldAnswer(tg.CallbackConfig{
			CallbackQueryID: cbQuery.ID,
			Text:            p.Sprintf("Language is %v now", c.State.Language),
		})

		if !c.State.IsAuthorized() {
			c.ShouldSend(c.newRawHTMLMessage(p.Sprintf("<welcome HTML>")))
			c.State.Expectation = store.ExpectAuthToken
			c.ShouldOK(c.saveState())
		} else {
			// TODO
		}
	} else if isEventAction(cbData) {
		eventRec, err := c.App.GetMsgRec(c.ID, msg.MessageID)
		if err != nil {
			text := emoji.Parse(p.Sprintf(":warning: Cannot load event: %v", err))
			if errors.Is(err, store.ErrNotFound) {
				text = emoji.Parse(p.Sprintf(":warning: Cannot load event data, probably this message is too old"))
			}
			c.ShouldAnswer(tg.CallbackConfig{
				CallbackQueryID: cbQuery.ID,
				Text:            text,
			})
			return
		}

		event := eventRec.Event

		if err := c.ShouldOK(event.LoadPost(c.frfAPI())); err != nil {
			c.ShouldAnswer(tg.CallbackConfig{
				CallbackQueryID: cbQuery.ID,
				Text:            emoji.Parse(p.Sprintf(":warning: FreeFeed error: %v", err)),
			})
			return
		}

		if (cbData == doReply || cbData == doReplyAt) && event.Post != nil {
			c.State.ClearExpectations()
			c.State.Expectation = store.ExpectComment
			c.State.ReactToMessageID = msg.MessageID
			if msg.ReplyToMessage != nil {
				c.State.ReactToMessageID = msg.ReplyToMessage.MessageID
			}
			if cbData == doReplyAt {
				c.State.CommentPrefix = event.CreatedUser.String() + " "
			}
			c.saveState()
			c.App.PauseEvents(c.ID)
			c.ShouldAnswer(tg.CallbackConfig{CallbackQueryID: cbQuery.ID})
		} else if cbData == doAcceptRequest || cbData == doRejectRequest {
			var err error
			if event.Group == nil {
				if cbData == doAcceptRequest {
					err = c.frfAPI().AcceptSubscriptionRequest(event.CreatedUser.Name)
				} else {
					err = c.frfAPI().RejectSubscriptionRequest(event.CreatedUser.Name)
				}
			} else {
				if cbData == doAcceptRequest {
					err = c.frfAPI().AcceptGroupSubscriptionRequest(event.CreatedUser.Name, event.Group.Name)
				} else {
					err = c.frfAPI().RejectGroupSubscriptionRequest(event.CreatedUser.Name, event.Group.Name)
				}
			}
			if err != nil {
				c.ShouldAnswer(tg.CallbackConfig{
					CallbackQueryID: cbQuery.ID,
					Text:            emoji.Parse(p.Sprintf(":warning: FreeFeed error: %v", err)),
				})
				return
			}

			newText := msg.Text + "\n\n"
			if cbData == doAcceptRequest {
				newText += emoji.Parse(p.Sprintf(":white_check_mark: Accepted!"))
			} else {
				newText += emoji.Parse(p.Sprintf(":x: Rejected!"))
			}
			msg := tg.NewEditMessageText(c.ID, msg.MessageID, newText)
			msg.ParseMode = "HTML"
			msg.DisableWebPagePreview = true
			msg.ReplyMarkup = &tg.InlineKeyboardMarkup{InlineKeyboard: [][]tg.InlineKeyboardButton{}}
			c.ShouldSend(msg)

		} else if cbData == doPostMore {
			msg := tg.NewEditMessageReplyMarkup(c.ID, msg.MessageID, c.postButtonsMore(event))
			c.ShouldSend(msg)

		} else if cbData == doPostBack {
			msg := tg.NewEditMessageReplyMarkup(c.ID, msg.MessageID, c.postButtons(event))
			c.ShouldSend(msg)

		} else if cbData == doTrackPost || cbData == doUntrackPost {
			var err error
			if cbData == doTrackPost {
				if err := c.ShouldOK(c.App.TrackPost(c.ID, event.PostID)); err == nil {
					// subscribe
					c.ShouldOK(c.App.RTSend(c.ID, "subscribe", types.UserSubsPayload{PostIDs: []uuid.UUID{event.PostID}}, nil))
				}
			} else {
				if err := c.ShouldOK(c.App.UntrackPost(c.ID, event.PostID)); err == nil {
					// unsubscribe
					c.ShouldOK(c.App.RTSend(c.ID, "unsubscribe", types.UserSubsPayload{PostIDs: []uuid.UUID{event.PostID}}, nil))
				}
			}
			if err != nil {
				c.ShouldAnswer(tg.CallbackConfig{
					CallbackQueryID: cbQuery.ID,
					Text:            emoji.Parse(p.Sprintf(":warning: Error: %v", err)),
				})
				return
			}
			msg := tg.NewEditMessageReplyMarkup(c.ID, msg.MessageID, c.postButtonsMore(event))
			c.ShouldSend(msg)

		} else {
			c.ShouldAnswer(tg.CallbackConfig{
				CallbackQueryID: cbQuery.ID,
				Text:            emoji.Parse(p.Sprintf(":alien: Unknown command %v", cbQuery.Data)),
			})
		}
	} else if cbData == "cancel" {
		c.State.ClearExpectations()
		c.saveState()
		c.App.ResumeEvents(c.ID)
		c.ShouldAnswer(tg.CallbackConfig{
			CallbackQueryID: cbQuery.ID,
			Text:            p.Sprintf("Action is cancelled"),
		})
	} else {
		// Unknown or irrelevant command
		c.errorLog().Printf("Unknown callback data received: %v", cbQuery.Data)
		c.ShouldAnswer(tg.CallbackConfig{
			CallbackQueryID: cbQuery.ID,
			Text:            emoji.Parse(p.Sprintf(":alien: Unknown command %v", cbQuery.Data)),
		})
	}
}
