package chat

import (
	"github.com/davidmz/freefeed-tg-client/store"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/kr/pretty"
	"golang.org/x/text/message"
)

func (c *Chat) handleMessage(update tg.Update) {
	msg := update.Message
	if msg == nil || msg.Command() != "" {
		return
	}

	p := message.NewPrinter(c.State.Language)

	if c.State.Expectation == store.ExpectAuthToken {
		token := msg.Text
		_, _, err := new(jwt.Parser).ParseUnverified(token, new(jwt.StandardClaims))
		if err != nil {
			c.debugLog().Printf("invalid token: %v", err)
			c.ShouldSend(c.newHTMLMessage(p.Sprintf("Looks like this token isn't valid.")))
			return
		}

		// Delete message with the token for safety
		c.Should(c.App.Tg().DeleteMessage(tg.DeleteMessageConfig{
			ChatID:    c.ID,
			MessageID: msg.MessageID,
		}))

		statusMsg, _ := c.ShouldSend(c.newHTMLMessage(p.Sprintf("Checking your token...")))

		user, err := c.frfAPIWithToken(token).GetMe()

		if err != nil {
			msg := tg.NewEditMessageText(c.ID, statusMsg.MessageID, p.Sprintf("Something wrong happened: %v", err))
			c.ShouldSend(msg)
		} else {
			msg := tg.NewEditMessageText(c.ID, statusMsg.MessageID, c.App.Linkify(p.Sprintf(
				"Hello, @%s!\nIt's all set. Now when the bot sees the update on FreeFeed, it will show it to you.",
				user.Name,
			)))
			msg.ParseMode = "HTML"
			c.ShouldSend(msg)

			c.State.ClearExpectations()
			c.State.UserID = user.ID
			c.State.AccessToken = token
			c.ShouldOK(c.saveState())

			c.App.StartRealtime(c.ID)
		}
	} else if c.State.Expectation == store.ExpectComment {
		if msg.Text == "" {
			pretty.Println(msg)
			c.ShouldSend(c.newHTMLMessage(p.Sprintf("Can not send a comment without a text")))
			return
		}

		eventRec, err := c.App.GetMsgRec(c.ID, c.State.ReactToMessageID)
		if err != nil {
			c.ShouldSend(c.newHTMLMessage(p.Sprintf("Error creating comment: %v", err)))
			return
		}

		event := eventRec.Event

		commentText := c.State.CommentPrefix + msg.Text
		comment, err := c.frfAPI().AddComment(event.PostID, commentText)
		if err != nil {
			c.ShouldSend(c.newHTMLMessage(p.Sprintf("Error creating comment: %v", err)))
			return
		}

		// Comment created
		msg := c.newHTMLMessage(p.Sprintf(":tada: Comment successfully created!"))
		msg.ReplyToMessageID = c.State.ReactToMessageID
		msg.ReplyMarkup = c.sentCommentButtons(event, comment.ID)
		c.ShouldSendAndSave(msg, store.SentMsgRec{Event: event, ReplyToID: msg.ReplyToMessageID})

		c.State.ClearExpectations()
		c.ShouldOK(c.saveState())
		c.App.ResumeEvents(c.ID)
	} else {

		if msg.ReplyToMessage != nil {
			eventRec, err := c.App.GetMsgRec(c.ID, msg.ReplyToMessage.MessageID)
			c.ShouldOK(err)
			if err == nil {
				// We have a reply to the event-related message
				event := eventRec.Event
				if event.PostID != uuid.Nil && msg.Text != "" {
					commentText := msg.Text
					comment, err := c.frfAPI().AddComment(event.PostID, commentText)
					if err != nil {
						c.ShouldSend(c.newHTMLMessage(p.Sprintf("Error creating comment: %v", err)))
						return
					}

					// Comment created
					msg := c.newHTMLMessage(p.Sprintf(":tada: Comment successfully created!"))
					msg.ReplyToMessageID = eventRec.ReplyToID
					msg.ReplyMarkup = c.sentCommentButtons(event, comment.ID)
					c.ShouldSendAndSave(msg, eventRec)

					c.State.ClearExpectations()
					c.ShouldOK(c.saveState())
					c.App.ResumeEvents(c.ID)
					return
				}
			}
		}

		pretty.Print(update)
		c.ShouldSend(c.newHTMLMessage(p.Sprintf(":shrug: Unknown command")))
	}
}
