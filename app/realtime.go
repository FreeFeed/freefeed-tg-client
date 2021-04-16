package app

import (
	"encoding/json"
	"fmt"

	"github.com/davidmz/freefeed-tg-client/chat"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/socketio"
	"github.com/davidmz/freefeed-tg-client/store"
	"github.com/davidmz/freefeed-tg-client/types"
	"github.com/davidmz/mustbe"
	"github.com/gofrs/uuid"
)

func (a *App) StartRealtime(chatID types.TgChatID) {
	a.rtConnLock.Lock()
	defer a.rtConnLock.Unlock()

	if rt, ok := a.rtConns[chatID]; ok {
		delete(a.rtConns, chatID)
		rt.Close()
	}

	rt := socketio.Open(
		"wss://"+a.FreeFeedHost+"/socket.io/?EIO=3&transport=websocket",
		socketio.WithLogger(a.DebugLogger.Fork(fmt.Sprintf("tg-client:rt:%d", chatID))),
	)
	a.rtConns[chatID] = rt

	a.waitGroup.Add(1)
	a.DebugLogger.Println("▶️ Starting RT loop for", chatID)
	go func() {
		defer a.waitGroup.Done()
		defer a.DebugLogger.Println("⏹️ Closing RT loop for", chatID)
		for {
			a.DebugLogger.Println("Waiting for select in RT loop of", chatID)
			select {
			case <-a.closeChan:
				a.DebugLogger.Println("Closing RT connection", chatID)
				a.StopRealtime(chatID)
				return
			case _, opened := <-rt.Connected():
				if !opened {
					// Connection is permanently closed
					a.DebugLogger.Println("RT connection is permanently closed", chatID)
					return
				}
				a.onRTConnect(chatID, rt)
			case msg := <-rt.Messages():
				a.onRTMessage(chatID, msg)
			}
		}
	}()
}

func (a *App) StopRealtime(chatID types.TgChatID) {
	a.rtConnLock.Lock()
	defer a.rtConnLock.Unlock()

	a.DebugLogger.Println("Trying to stop RT connection", chatID)
	if rt, ok := a.rtConns[chatID]; ok {
		delete(a.rtConns, chatID)
		rt.Close()
	}
}

func (a *App) onRTConnect(chatID types.TgChatID, rt *socketio.Connection) {

	logger := a.DebugLogger.Fork(fmt.Sprintf("tg-client:rt:%d", chatID))

	defer mustbe.Catched(func(err error) {
		logger.Println("Cannot process connect:", err)
	})

	logger.Println("RT Connected!")
	defer logger.Println("Finish connect procedure")

	state := mustbe.OKVal(a.Store.LoadState(chatID)).(*store.State)

	// Authorize connection
	reply := mustbe.OKVal(rt.Send("auth", authTokenPayload{state.AccessToken})).([]byte)
	logger.Println("Auth reply:", string(reply))

	tracked := mustbe.OKVal(a.TrackedEntities(chatID)).(store.TrackedEntities)
	reply = mustbe.OKVal(rt.Send(
		"subscribe",
		types.UserSubsPayload{
			UserIDs: []uuid.UUID{state.UserID},
			PostIDs: tracked.PostIDs,
		},
	)).([]byte)
	logger.Println("Subscribe reply:", string(reply))
}

func (a *App) onRTMessage(chatID types.TgChatID, msg socketio.IncomingMessage) {
	defer mustbe.Catched(func(err error) {
		a.ErrorLogger.Println("Cannot process message:", err)
	})

	a.DebugLogger.Println("Start process RT message of type", msg.Type)
	defer a.DebugLogger.Println("Finish process RT message of type", msg.Type)

	var events frf.Events

	if msg.Type == "event:new" {
		mustbe.OK(json.Unmarshal(msg.Payload, &events))
	} else if msg.Type == "comment:new" {
		var cEvent frf.NewCommentEvent
		mustbe.OK(json.Unmarshal(msg.Payload, &cEvent))

		// Make a fake event
		event := &frf.Event{
			Type:          "__" + msg.Type,
			CommentID:     cEvent.Comments.ID,
			PostID:        cEvent.Comments.PostID,
			CreatedUserID: cEvent.Comments.CreatedBy,
		}
		for _, u := range cEvent.Users {
			if u.ID == event.CreatedUserID {
				event.CreatedUser = u
				break
			}
		}

		events = frf.Events{event}
	} else {
		return
	}

	ch := mustbe.OKVal(chat.New(chatID, a)).(*chat.Chat)
	ch.ProcessEvents(events)
}

type authTokenPayload struct {
	AuthToken string `json:"authToken"`
}
