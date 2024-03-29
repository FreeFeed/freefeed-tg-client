package chat

import (
	"github.com/FreeFeed/freefeed-tg-client/frf"
	"github.com/FreeFeed/freefeed-tg-client/store"
	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/davidmz/debug-log"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ID = types.TgChatID

type App interface {
	store.Store

	DebugLog() debug.Logger
	ErrorLog() debug.Logger
	FreeFeedAPI() *frf.API
	Tg() *tg.BotAPI
	Send(tg.Chattable) (tg.Message, error)
	Linkify(string) string
	ContentOf(string) string

	StartRealtime(ID)
	StopRealtime(ID)

	EventsPaused(ID) bool
	PauseEvents(ID)
	ResumeEvents(ID)

	RTSend(chatID types.TgChatID, cmd string, payload interface{}, reply interface{}) error
}
