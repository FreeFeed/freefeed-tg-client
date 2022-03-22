package store

import (
	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/gofrs/uuid"
	"golang.org/x/text/language"
)

type Expectation string

const (
	ExpectNothing   Expectation = ""
	ExpectLanguage  Expectation = "lang"
	ExpectAuthToken Expectation = "token"
	ExpectComment   Expectation = "comment"
)

// State is the saved state of a chat.
type State struct {
	ID          types.TgChatID
	Language    language.Tag
	UserID      uuid.UUID
	AccessToken string
	LastEventID uuid.UUID
	Expectation Expectation

	ReactToMessageID int
	CommentToPostID  uuid.UUID
	CommentPrefix    string
}

// IsAuthorized returns true if the user is authorized.
func (s *State) IsAuthorized() bool {
	return s.AccessToken != ""
}

func (s *State) ClearExpectations() {
	s.Expectation = ""
	s.ReactToMessageID = 0
	s.CommentToPostID = uuid.Nil
	s.CommentPrefix = ""
}

func (s *State) IsPausedExpectation() bool {
	return s.Expectation == ExpectComment
}
