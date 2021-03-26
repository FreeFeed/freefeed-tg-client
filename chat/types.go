package chat

import (
	"github.com/davidmz/debug-log"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/store"
	"github.com/davidmz/freefeed-tg-client/types"
	tg "github.com/davidmz/telegram-bot-api"
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

	StartRealtime(ID)
	StopRealtime(ID)

	EventsPaused(ID) bool
	PauseEvents(ID)
	ResumeEvents(ID)

	RTSend(chatID types.TgChatID, cmd string, payload interface{}, reply interface{}) error
}
