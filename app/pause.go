package app

import (
	"encoding/json"

	"github.com/davidmz/freefeed-tg-client/chat"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/types"
	"github.com/davidmz/mustbe"
)

func (a *App) EventsPaused(chatID types.TgChatID) bool { return a.pauseManager.IsPaused(chatID) }
func (a *App) PauseEvents(chatID types.TgChatID)       { a.pauseManager.Pause(chatID) }
func (a *App) ResumeEvents(chatID types.TgChatID)      { a.pauseManager.Resume(chatID) }

func (a *App) doResumeEvents(chatID types.TgChatID) {
	defer mustbe.Catched(func(err error) {
		a.ErrorLogger.Println("Cannot resume events:", err)
	})

	ch := mustbe.OKVal(chat.New(chatID, a)).(*chat.Chat)

	entries := mustbe.OKVal(a.Store.LoadAndDeleteQueue(chatID)).([]json.RawMessage)
	a.DebugLogger.Printf("Loaded %d events for %v", len(entries), chatID)

	var events []*frf.Event
	for _, entry := range entries {
		var event frf.Event
		if err := json.Unmarshal(entry, &event); err != nil {
			a.ErrorLogger.Println("Cannot restore event:", err)
			continue
		}
		events = append(events, &event)
	}
	a.DebugLogger.Printf("Events parsed for %v", chatID)

	if ch.State.IsPausedExpectation() {
		ch.State.ClearExpectations()
		mustbe.OK(a.Store.SaveState(ch.State))
	}

	ch.ProcessEvents(events)
}
