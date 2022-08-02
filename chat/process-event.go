package chat

import (
	"encoding/json"

	"github.com/FreeFeed/freefeed-tg-client/frf"
	"github.com/FreeFeed/freefeed-tg-client/store"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/gofrs/uuid"
	"golang.org/x/text/message"
)

var mutedEvents = []string{
	"banned_user",
	"unbanned_user",
	"group_created",
}

func isMutedEvent(event *frf.Event) bool {
	for _, m := range mutedEvents {
		if m == event.Type {
			return true
		}
	}
	return false
}

func (c *Chat) ProcessEvents(events []*frf.Event) {
	c.debugLog().Printf("Start ProcessEvents for %d events", len(events))
	defer c.debugLog().Printf("Finish ProcessEvents for %d events", len(events))

	c.debugLog().Printf("Checking paused state...")
	isPaused := c.App.EventsPaused(c.ID)
	c.debugLog().Printf("Result: %v", isPaused)

	for _, event := range events {
		c.debugLog().Printf("ProcessEvents for %s", event.Type)
		if isMutedEvent(event) {
			c.debugLog().Printf("Event %s is muted", event.Type)
			continue
		}

		if isPaused {
			c.debugLog().Printf("Paused, adding %s to event queue", event.Type)
			data, _ := c.Should(json.Marshal(event))
			c.ShouldOK(c.App.AddToQueue(c.ID, data.([]byte)))
		} else if msg := c.renderEvent(event); msg != nil {
			c.debugLog().Printf("Sending %s to user", event.Type)
			c.ShouldSendAndSave(msg, store.SentMsgRec{Event: event})
		}
	}
}

func (c *Chat) renderEvent(event *frf.Event) tg.Chattable {
	c.debugLog().Println("Start renderEvent for", event.Type)
	defer c.debugLog().Println("Finish renderEvent for", event.Type)

	p := message.NewPrinter(c.State.Language)
	event.LoadPost(c.frfAPI())

	switch event.Type {
	// ===========================
	// Posts and comments
	// ===========================
	case "mention_in_post":
		if event.Post != nil && event.Post.IsDirect() {
			// We will receive this post in 'direct' event
			return nil
		}

		text := p.Sprintf(":e-mail: %s mentioned you in the post:", event.CreatedUser)
		if event.Group != nil {
			text = p.Sprintf(
				":e-mail: %s mentioned you in the post in %s:",
				event.CreatedUser,
				event.Group,
			)
		}
		return c.withPostBody(c.newHTMLMessage(text), event)
	case "mention_in_comment":
		if event.Post != nil && event.Post.IsDirect() {
			// We will receive this in 'direct_comment' event
			return nil
		}

		if ok, _ := c.App.IsPostTracked(c.ID, event.PostID); ok {
			// We will receive this with post subscription
			return nil
		}

		headText := p.Sprintf(
			":e-mail: %s mentioned you in a comment to the post \"%s\":",
			event.CreatedUser,
			c.App.ContentOf(event.Post.Digest()),
		)
		if event.Group != nil {
			headText = p.Sprintf(
				":e-mail: %s mentioned you in a comment to the post in %s \"%s\":",
				event.CreatedUser,
				event.Group,
				c.App.ContentOf(event.Post.Digest()),
			)
		}

		return c.withCommentBody(c.newHTMLMessage(headText), event)
	case "mention_comment_to":
		if event.Post != nil && event.Post.IsDirect() {
			// We will receive this in 'direct_comment' event
			return nil
		}

		if ok, _ := c.App.IsPostTracked(c.ID, event.PostID); ok {
			// We will receive this with post subscription
			return nil
		}

		headText := p.Sprintf(
			":e-mail: %s replied to you in a comment to the post \"%s\":",
			event.CreatedUser,
			c.App.ContentOf(event.Post.Digest()),
		)
		if event.Group != nil {
			headText = p.Sprintf(
				":e-mail: %s replied to you in a comment to the post in %s \"%s\":",
				event.CreatedUser,
				event.Group,
				c.App.ContentOf(event.Post.Digest()),
			)
		}

		return c.withCommentBody(c.newHTMLMessage(headText), event)

	case "backlink_in_post":
		if ok, _ := c.App.IsPostTracked(c.ID, event.PostID); ok {
			// We will receive this with post subscription
			return nil
		}

		headText := ""
		if event.RefCommentID != uuid.Nil {
			headText = p.Sprintf(
				":link: %s mentioned your comment in the post:",
				event.CreatedUser,
			)
			if event.Group != nil {
				headText = p.Sprintf(
					":link: %s mentioned your comment in the post in %s:",
					event.CreatedUser,
					event.Group,
				)
			}
		} else {
			headText = p.Sprintf(
				":link: %s mentioned your post in the post:",
				event.CreatedUser,
			)
			if event.Group != nil {
				headText = p.Sprintf(
					":link: %s mentioned your post in the post in %s:",
					event.CreatedUser,
					event.Group,
				)
			}
		}
		return c.withPostBody(c.newHTMLMessage(headText), event)

	case "backlink_in_comment":
		if ok, _ := c.App.IsPostTracked(c.ID, event.PostID); ok {
			// We will receive this with post subscription
			return nil
		}

		headText := ""
		if event.RefCommentID != uuid.Nil {
			headText = p.Sprintf(
				":link: %s mentioned your comment in the comment to post \"%s\":",
				event.CreatedUser,
				c.App.ContentOf(event.Post.Digest()),
			)
		} else {
			headText = p.Sprintf(
				":link: %s mentioned your post in the comment to post \"%s\":",
				event.CreatedUser,
				c.App.ContentOf(event.Post.Digest()),
			)
		}
		return c.withCommentBody(c.newHTMLMessage(headText), event)

	case "direct_left":
		if event.CreatedUser.ID == c.State.UserID {
			// Our action
			return nil
		}
		headText := p.Sprintf(
			":door: %s left the direct message \"%s\":",
			event.CreatedUser,
			c.App.ContentOf(event.Post.Digest()),
		)
		msg := c.newHTMLMessage(headText)
		msg.ReplyMarkup = c.postButtons(event)
		return msg

	case "direct":
		headText := p.Sprintf(":e-mail: You received a direct message from %s:", event.CreatedUser)
		return c.withPostBody(c.newHTMLMessage(headText), event)
	case "direct_comment":
		if ok, _ := c.App.IsPostTracked(c.ID, event.PostID); ok {
			// We will receive this with post subscription
			return nil
		}

		headText := p.Sprintf(
			":e-mail: New comment was posted by %s to the direct message \"%s\":",
			event.CreatedUser,
			c.App.ContentOf(event.Post.Digest()),
		)
		return c.withCommentBody(c.newHTMLMessage(headText), event)

	case "__comment:new":
		if event.CreatedUser == nil {
			// Hidden comment, don't do anything
			return nil
		}
		if event.CreatedUser.ID == c.State.UserID {
			// Comment from ourselves
			return nil
		}
		headText := p.Sprintf(
			":e-mail: New comment was posted by %s to the post \"%s\":",
			event.CreatedUser,
			c.App.ContentOf(event.Post.Digest()),
		)
		return c.withCommentBody(c.newHTMLMessage(headText), event)

	// ===========================
	// Incoming subscription requests
	// ===========================
	case "subscription_requested":
		msg := c.newHTMLMessage(p.Sprintf(":raising_hand: %s sent you a subscription request", event.CreatedUser))
		msg.ReplyMarkup = c.subscrButtons(event)
		return msg
	case "group_subscription_requested":
		msg := c.newHTMLMessage(p.Sprintf(":raising_hand: %s sent a request to join %s that you admin", event.CreatedUser, event.Group))
		msg.ReplyMarkup = c.subscrButtons(event)
		return msg

		// ===========================
		// Outcoming subscription requests
		// ===========================
	case "subscription_request_approved":
		text := p.Sprintf(`:white_check_mark: Your subscription request to %s was approved`, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "subscription_request_rejected":
		text := p.Sprintf(`:no_entry_sign: Your subscription request to %s was rejected`, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "group_subscription_approved":
		text := p.Sprintf(`:white_check_mark: Your request to join group %s was approved`, event.Group)
		return c.newHTMLMessage(text)
	case "group_subscription_rejected":
		text := p.Sprintf(`:no_entry_sign: Your request to join group %s was rejected`, event.Group)
		return c.newHTMLMessage(text)

	// ===========================
	// Your subscribers
	// ===========================
	case "user_subscribed":
		text := p.Sprintf(`:plus: %s subscribed to your feed`, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "user_unsubscribed":
		text := p.Sprintf(`:minus: %s unsubscribed from your feed`, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "group_subscribed":
		text := p.Sprintf(`:plus: %s subscribed to %s`, event.CreatedUser, event.Group)
		return c.newHTMLMessage(text)
	case "group_unsubscribed":
		text := p.Sprintf(`:minus: %s unsubscribed from %s`, event.CreatedUser, event.Group)
		return c.newHTMLMessage(text)
	case "subscription_request_revoked":
		text := p.Sprintf(`:minus: %s revoked subscription request to you`, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "group_subscription_request_revoked":
		text := p.Sprintf(`:minus: %s revoked subscription request to %s`, event.CreatedUser, event.Group)
		return c.newHTMLMessage(text)

		// ===========================
		// Group moderation
		// ===========================
	case "group_admin_promoted":
		if event.CreatedUser.ID == c.State.UserID {
			// We initiated the event ourselves
			return nil
		}
		text := p.Sprintf(`:plus: %s promoted %s to admin in the group %s`,
			event.CreatedUser, event.AffectedUser, event.Group)
		return c.newHTMLMessage(text)
	case "group_admin_demoted":
		if event.CreatedUser.ID == c.State.UserID {
			// We initiated the event ourselves
			return nil
		}
		text := p.Sprintf(`:minus: %s revoked admin privileges from %s in the group %s`,
			event.CreatedUser, event.AffectedUser, event.Group)
		return c.newHTMLMessage(text)
	case "managed_group_subscription_approved":
		if event.CreatedUser.ID == c.State.UserID {
			// We initiated the event ourselves
			return nil
		}
		text := p.Sprintf(`:plus: %s request to join %s was approved by %s`,
			event.AffectedUser, event.Group, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "managed_group_subscription_rejected":
		if event.CreatedUser.ID == c.State.UserID {
			// We initiated the event ourselves
			return nil
		}
		text := p.Sprintf(`:minus: %s request to join %s was rejected by %s`,
			event.AffectedUser, event.Group, event.CreatedUser)
		return c.newHTMLMessage(text)
	case "comment_moderated":
		createdUserStr := p.Sprintf("group admin")
		if event.CreatedUser != nil {
			createdUserStr = event.CreatedUser.String()
		}
		text := p.Sprintf(
			":cop: %s has deleted your comment to the \"%s\":",
			createdUserStr,
			c.App.ContentOf(event.Post.Digest()),
		)
		if event.Group != nil {
			text = p.Sprintf(
				":cop: %s has deleted your comment to the post in %s \"%s\":",
				createdUserStr,
				event.Group,
				c.App.ContentOf(event.Post.Digest()),
			)
		}

		msg := c.newHTMLMessage(text)
		msg.ReplyMarkup = c.postButtons(event)

		return msg
	case "comment_moderated_by_another_admin":
		text := p.Sprintf(
			":cop: %s has removed a comment from %s to the post in the group %s \"%s\":",
			event.CreatedUser,
			event.AffectedUser,
			event.Group,
			c.App.ContentOf(event.Post.Digest()),
		)

		msg := c.newHTMLMessage(text)
		msg.ReplyMarkup = c.postButtons(event)

		return msg
	case "post_moderated":
		createdUserStr := p.Sprintf("group admin")
		if event.CreatedUser != nil {
			createdUserStr = event.CreatedUser.String()
		}
		text := p.Sprintf(
			":cop: %s has removed your post from the group %s",
			createdUserStr,
			event.Group,
		)

		if event.Post != nil {
			text = p.Sprintf(
				":cop: %s has removed your post from the group %s \"%s\":",
				createdUserStr,
				event.Group,
				c.App.ContentOf(event.Post.Digest()),
			)
		}

		msg := c.newHTMLMessage(text)
		if event.Post != nil {
			msg.ReplyMarkup = c.postButtons(event)
		}

		return msg
	case "post_moderated_by_another_admin":
		text := p.Sprintf(
			":cop: %s has removed the post from %s from the group %s",
			event.CreatedUser,
			event.AffectedUser,
			event.Group,
		)

		if event.Post != nil {
			text = p.Sprintf(
				":cop: %s has removed the post from %s from the group %s \"%s\":",
				event.CreatedUser,
				event.AffectedUser,
				event.Group,
				c.App.ContentOf(event.Post.Digest()),
			)
		}

		msg := c.newHTMLMessage(text)
		if event.Post != nil {
			msg.ReplyMarkup = c.postButtons(event)
		}

		return msg

	case "blocked_in_group":
		who := p.Sprintf("Group admin")
		if event.CreatedUser != nil {
			who = event.CreatedUser.String()
		}
		whom := p.Sprintf("you")
		if event.AffectedUser.ID != c.State.UserID {
			who = event.AffectedUser.String()
		}
		text := p.Sprintf(
			":cop: %s blocked %s in group %s",
			who, whom, event.Group)
		return c.newHTMLMessage(text)

	case "unblocked_in_group":
		who := p.Sprintf("Group admin")
		if event.CreatedUser != nil {
			who = event.CreatedUser.String()
		}
		whom := p.Sprintf("you")
		if event.AffectedUser.ID != c.State.UserID {
			who = event.AffectedUser.String()
		}
		text := p.Sprintf(
			":cop: %s unblocked %s in group %s",
			who, whom, event.Group)
		return c.newHTMLMessage(text)

	// ===========================
	// Misc
	// ===========================
	case "invitation_used":
		text := p.Sprintf(`:tada: %s has joined FreeFeed using your invitation`, event.CreatedUser)
		return c.newHTMLMessage(text)

	default:
		return c.newHTMLMessage(p.Sprintf(":alien: Unknown event: %v", event.Type))
	}
}

const bodySeparator = "\n\n"

func (c *Chat) withPostBody(msg *tg.MessageConfig, event *frf.Event) (out tg.Chattable) {
	if event.PostID == uuid.Nil {
		return msg
	}

	if err := event.LoadPost(c.frfAPI()); err != nil {
		msg.Text += bodySeparator + err.Error()
	}

	msg.Text += bodySeparator + c.App.ContentOf(c.App.Linkify(event.Post.Body))
	msg.ReplyMarkup = c.postButtons(event)
	return msg
}

func (c *Chat) withCommentBody(msg *tg.MessageConfig, event *frf.Event) (out tg.Chattable) {
	if event.PostID == uuid.Nil {
		return msg
	}

	if err := event.LoadPost(c.frfAPI()); err != nil {
		msg.Text += bodySeparator + err.Error()
		msg.ReplyMarkup = c.postButtons(event)
		return msg
	}

	comment := frf.Comment{Body: "Comment not found"}
	for _, c := range event.Post.Comments {
		if event.CommentID == c.ID {
			comment = c
			break
		}
	}

	msg.Text += bodySeparator + c.App.ContentOf(c.App.Linkify(comment.Body))
	msg.ReplyMarkup = c.postButtons(event)
	return msg
}
