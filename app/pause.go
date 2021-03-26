package app

import (
	"encoding/json"
	"time"

	"github.com/davidmz/freefeed-tg-client/chat"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/types"
	"github.com/davidmz/mustbe"
)

// pauseInterval is interval for pause when the user creates a new comment or post
const pauseInterval = 20 * time.Minute

func (a *App) EventsPaused(chatID types.TgChatID) bool {
	a.pausedChatsLock.RLock()
	defer a.pausedChatsLock.RUnlock()

	_, ok := a.pausedChats[chatID]
	return ok
}

func (a *App) PauseEvents(chatID types.TgChatID) {
	a.pausedChatsLock.Lock()
	defer a.pausedChatsLock.Unlock()

	// Already paused?
	if t, ok := a.pausedChats[chatID]; ok {
		a.DebugLogger.Println("Already paused:", chatID)
		delete(a.pausedChats, chatID)
		if !t.Stop() {
			<-t.C
		}
		a.DebugLogger.Println("Prev pause cancelled:", chatID)
	}

	a.DebugLogger.Println("Schedule resume for:", chatID)
	a.pausedChats[chatID] = time.AfterFunc(pauseInterval, func() { a.ResumeEvents(chatID) })
}

func (a *App) ResumeEvents(chatID types.TgChatID) {
	defer mustbe.Catched(func(err error) {
		a.ErrorLogger.Println("Cannot resume events:", err)
	})

	a.pausedChatsLock.Lock()
	defer a.pausedChatsLock.Unlock()

	if t, ok := a.pausedChats[chatID]; ok {
		a.DebugLogger.Println("Already paused:", chatID)
		delete(a.pausedChats, chatID)
		if !t.Stop() {
			<-t.C
		}
		a.DebugLogger.Println("Prev pause cancelled:", chatID)
	} else {
		// nothing to do
		return
	}

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

	// Processing in separated thread because of n.pausedChatsLock
	go ch.ProcessEvents(events)
}
