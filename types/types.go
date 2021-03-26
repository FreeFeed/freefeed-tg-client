package types

import (
	"errors"

	"github.com/gofrs/uuid"
)

// TgChatID is a type of Telegram chat ID
type TgChatID = int64

var ErrNotFound = errors.New("not found")

type UserSubsPayload struct {
	UserIDs []uuid.UUID `json:"user,omitempty"`
	PostIDs []uuid.UUID `json:"post,omitempty"`
}
