package store

import (
	"encoding/json"
	"errors"

	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/gofrs/uuid"
)

// Store abstracts database interface.
type Store interface {
	// LoadState returnd state by chatID or nil if there is no state
	LoadState(chatID types.TgChatID) (*State, error)
	SaveState(state *State) error
	DeleteState(chatID types.TgChatID) error
	ListIDs() ([]types.TgChatID, error)

	// EventsQueue
	AddToQueue(chatID types.TgChatID, entry json.RawMessage) error
	LoadAndDeleteQueue(chatID types.TgChatID) ([]json.RawMessage, error)

	// EventsStore
	GetMsgRec(chatID types.TgChatID, messageID int) (SentMsgRec, error)
	PutMsgRec(chatID types.TgChatID, rec SentMsgRec) error

	// Tracked posts
	TrackPost(chatID types.TgChatID, postID uuid.UUID) error
	UntrackPost(chatID types.TgChatID, postID uuid.UUID) error
	IsPostTracked(chatID types.TgChatID, postID uuid.UUID) (bool, error)
	TrackedEntities(chatID types.TgChatID) (TrackedEntities, error)
}

func NewChatState(chatID types.TgChatID) *State {
	return &State{
		ID: chatID,
	}
}

var ErrNotFound = errors.New("not found")
