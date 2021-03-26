package chat

import (
	"fmt"

	"github.com/davidmz/freefeed-tg-client/frf"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/enescakir/emoji"
	"github.com/gofrs/uuid"
	"golang.org/x/text/message"
)

func (c *Chat) postButtons(event *frf.Event) tg.InlineKeyboardMarkup {
	p := message.NewPrinter(c.State.Language)

	postLinkBtn := tg.NewInlineKeyboardButtonURL(
		emoji.Parse(p.Sprintf(":globe_with_meridians: Open post")),
		fmt.Sprintf("https://%s/posts/%s", c.frfAPI().HostName, event.PostID),
	)
	if event.CommentID != uuid.Nil {
		postLinkBtn = tg.NewInlineKeyboardButtonURL(
			emoji.Parse(p.Sprintf(":globe_with_meridians: Open comment")),
			fmt.Sprintf("https://%s/posts/%s#comment-%s",
				c.frfAPI().HostName, event.PostID, event.CommentID),
		)
	}

	if event.Post == nil {
		return tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{postLinkBtn})
	}

	row := []tg.InlineKeyboardButton{
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":speech_balloon: Reply")),
			doReply,
		),
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":speech_balloon: @-Reply")),
			doReplyAt,
		),
		postLinkBtn,
	}

	if !event.Post.IsDirect() {
		row = append(row, tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf("More\u2026")),
			doPostMore,
		))
	}

	return tg.NewInlineKeyboardMarkup(row)
}

func (c *Chat) postButtonsMore(event *frf.Event) tg.InlineKeyboardMarkup {
	p := message.NewPrinter(c.State.Language)

	row := []tg.InlineKeyboardButton{
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":back: Back")),
			doPostBack,
		),
	}

	if event.Post != nil && !event.Post.IsDirect() {
		ok, err := c.Should(c.App.IsPostTracked(c.ID, event.PostID))
		if err == nil {
			if ok.(bool) {
				row = append(row, tg.NewInlineKeyboardButtonData(
					emoji.Parse(p.Sprintf(":no_bell: Unsubscribe from comments")),
					doUntrackPost,
				))
			} else {
				row = append(row, tg.NewInlineKeyboardButtonData(
					emoji.Parse(p.Sprintf(":bell: Subscribe to comments")),
					doTrackPost,
				))
			}
		}
	}

	return tg.NewInlineKeyboardMarkup(row)
}

func (c *Chat) sentCommentButtons(event *frf.Event, commentID uuid.UUID) tg.InlineKeyboardMarkup {
	p := message.NewPrinter(c.State.Language)

	return tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":speech_balloon: Comment more")),
			doReply,
		),
		tg.NewInlineKeyboardButtonURL(
			emoji.Parse(p.Sprintf(":globe_with_meridians: Open comment")),
			fmt.Sprintf("https://%s/posts/%s#comment-%s",
				c.frfAPI().HostName, event.PostID, commentID),
		),
	})
}

func (c *Chat) subscrButtons(event *frf.Event) tg.InlineKeyboardMarkup {
	p := message.NewPrinter(c.State.Language)
	return tg.NewInlineKeyboardMarkup([]tg.InlineKeyboardButton{
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":white_check_mark: Accept")),
			doAcceptRequest,
		),
		tg.NewInlineKeyboardButtonData(
			emoji.Parse(p.Sprintf(":x: Reject")),
			doRejectRequest,
		),
	})
}
