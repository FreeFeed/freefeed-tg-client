package frf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gofrs/uuid"
)

const (
	DirectsFeedName = "Directs"
)

// User is a FreeFeed user
type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"username"`
	Type string    `json:"type"`
}

func (u *User) String() string {
	return "@" + u.Name
}

type Event struct {
	ID           uuid.UUID `json:"eventId"`
	Type         string    `json:"event_type"`
	CommentID    uuid.UUID `json:"comment_id"`
	PostID       uuid.UUID `json:"post_id"`
	RefCommentID uuid.UUID `json:"ref_comment_id"`
	RefPostID    uuid.UUID `json:"ref_post_id"`

	AffectedUserID uuid.UUID `json:"affected_user_id"`
	CreatedUserID  uuid.UUID `json:"created_user_id"`
	GroupID        uuid.UUID `json:"group_id"`
	PostAuthorID   uuid.UUID `json:"post_author_id"`

	AffectedUser *User
	CreatedUser  *User
	Group        *User
	PostAuthor   *User
	Post         *Post `json:"-"`
}

func (e *Event) LoadPost(api *API) error {
	if e.PostID == uuid.Nil {
		return nil // fmt.Errorf("PostID is empty")
	}
	if e.Post != nil {
		// Already loaded
		return nil
	}
	var err error
	e.Post, err = api.GetPost(e.PostID)
	if err != nil {
		return fmt.Errorf("cannot load post: %w", err)
	}
	return nil
}

type Feed struct {
	ID      uuid.UUID
	Name    string
	OwnerID uuid.UUID `json:"user"`
}

type Post struct {
	ID         uuid.UUID
	Body       string
	Recipients []Feed
	Comments   []Comment `json:"-"`
}

type Comment struct {
	ID   uuid.UUID
	Body string
}

func (p *Post) InNamedFeedOf(name string, ownerID uuid.UUID) bool {
	for _, f := range p.Recipients {
		if f.Name == name && f.OwnerID == ownerID {
			return true
		}
	}
	return false
}

func (p *Post) IsDirect() bool {
	for _, f := range p.Recipients {
		if f.Name == DirectsFeedName {
			return true
		}
	}
	return false
}

var whiteSpacesRe = regexp.MustCompile(`\s+`)

func (p *Post) Digest() string {
	const maxLen = 40
	words := whiteSpacesRe.Split(p.Body, -1)
	cutIdx := len(words)
	sumLen := 0
	for i, w := range words {
		sumLen += utf8.RuneCountInString(w) + 1
		if sumLen > maxLen {
			cutIdx = i + 1
			break
		}
	}
	s := strings.Join(words[:cutIdx], " ")
	if cutIdx < len(words) {
		s = s + "\u2026"
	}
	return s
}

// Error is the API error
type Error struct {
	Err            string
	HTTPStatus     string `json:"-"`
	HTTPStatusCode int    `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != "" {
		return e.Err
	}
	return e.HTTPStatus
}

func (e *Error) String() string { return e.Error() }

func errorFromResponse(resp *http.Response) error {
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}
	er := &Error{}
	data, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(data, er)
	er.HTTPStatus = resp.Status
	er.HTTPStatusCode = resp.StatusCode
	return er
}

// NotificationsResponse is a response of `GET /v2/notifications` request or the
// payload of the `event:new` message.
type NotificationsResponse struct {
	Events []*Event `json:"Notifications"`
	Groups []*User
	Users  []*User
}

// Events is a list of events.
type Events []*Event

// UnmarshalJSON implements the json.Unmarshaler interface for the Events.
func (e *Events) UnmarshalJSON(data []byte) error {
	resp := &struct {
		Notifications []*Event
		Groups        []*User
		Users         []*User
	}{}

	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}

	accByID := make(map[uuid.UUID]*User)
	for _, a := range resp.Users {
		accByID[a.ID] = a
	}
	for _, a := range resp.Groups {
		accByID[a.ID] = a
	}

	for _, n := range resp.Notifications {
		n.AffectedUser = accByID[n.AffectedUserID]
		n.CreatedUser = accByID[n.CreatedUserID]
		n.Group = accByID[n.GroupID]
		n.PostAuthor = accByID[n.PostAuthorID]
	}

	*e = resp.Notifications

	return nil
}

type addCommentRequest struct {
	Comment struct {
		PostID uuid.UUID `json:"postId"`
		Body   string    `json:"body"`
	} `json:"comment"`
}

func newAddCommentRequest(postID uuid.UUID, body string) *addCommentRequest {
	req := new(addCommentRequest)
	req.Comment.PostID = postID
	req.Comment.Body = body
	return req
}

type NewCommentEvent struct {
	Comments struct {
		ID        uuid.UUID
		PostID    uuid.UUID
		CreatedBy uuid.UUID
	}
	Users []*User
}
