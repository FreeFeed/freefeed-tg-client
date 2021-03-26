package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/davidmz/debug-log"
	"github.com/davidmz/freefeed-tg-client/chat"
	"github.com/davidmz/freefeed-tg-client/frf"
	"github.com/davidmz/freefeed-tg-client/socketio"
	"github.com/davidmz/freefeed-tg-client/store"
	"github.com/davidmz/freefeed-tg-client/types"
	tg "github.com/davidmz/telegram-bot-api"
	"github.com/enescakir/emoji"
)

var internaErrorMsg = emoji.Parse(":stop_sign: An internal error occurred during your request. If it repeats, please contact support.")

type App struct {
	store.Store
	FreeFeedHost string
	UserAgent    string
	DebugLogger  debug.Logger
	ErrorLogger  debug.Logger
	TgAPI        *tg.BotAPI

	updChannel tg.UpdatesChannel
	stateCache gcache.Cache
	waitGroup  sync.WaitGroup
	closeChan  chan struct{}

	rtConnLock sync.Mutex
	rtConns    map[types.TgChatID]*socketio.Connection

	pausedChats     map[types.TgChatID]*time.Timer
	pausedChatsLock sync.RWMutex
}

func (a *App) DebugLog() debug.Logger { return a.DebugLogger }
func (a *App) ErrorLog() debug.Logger { return a.ErrorLogger }
func (a *App) FreeFeedAPI() *frf.API {
	return &frf.API{HostName: a.FreeFeedHost, UserAgent: a.UserAgent}
}
func (a *App) Tg() *tg.BotAPI { return a.TgAPI }

// Start initializes the bot and starts the internal loops. This function doesnt
// return until the cxt is cancelled.
func (a *App) Start() (err error) {
	a.stateCache = gcache.
		New(1000).
		ARC().
		LoaderFunc(func(key interface{}) (interface{}, error) {
			state := store.NewChatState(key.(types.TgChatID))
			state.Expectation = store.ExpectLanguage
			return state, nil
		}).
		Build()

	a.closeChan = make(chan struct{})

	a.rtConns = make(map[types.TgChatID]*socketio.Connection)
	a.pausedChats = make(map[types.TgChatID]*time.Timer)

	a.updChannel, err = a.TgAPI.GetUpdatesChan(tg.UpdateConfig{Offset: 0, Timeout: 60})
	if err != nil {
		return
	}

	a.waitGroup.Add(1)
	a.DebugLogger.Println("▶️ Starting Telegram listener")
	go a.listenTelegram()

	// Starting realtime connections for existing users
	chatIDs, err := a.Store.ListIDs()
	if err != nil {
		a.ErrorLogger.Println("Cannot read chat IDs:", err)
		return err
	}
	a.DebugLogger.Println("Chat IDs found:", chatIDs)

	for _, chatID := range chatIDs {
		state, err := a.Store.LoadState(chatID)
		if err != nil {
			a.ErrorLogger.Println("Cannot load state:", err)
			return err
		}
		state.ClearExpectations()
		if err := a.SaveState(state); err != nil {
			a.ErrorLogger.Println("Cannot save state:", err)
			return err
		}

		a.StartRealtime(chatID)
	}

	// Waiting for finish
	a.waitGroup.Wait()
	return nil
}

func (a *App) Close() {
	close(a.closeChan)
}

func (a *App) listenTelegram() {
	defer a.waitGroup.Done()
	defer a.DebugLogger.Println("⏹️ Closing Telegram listener")
	for {
		select {
		case update := <-a.updChannel:
			a.waitGroup.Add(1)
			a.DebugLogger.Println("▶️ Starting TG update handler", update.UpdateID)
			go a.handleTgUpdate(update)
		case <-a.closeChan:
			a.DebugLogger.Println("Stop Telegram listener")
			return
		}
	}
}

func (a *App) handleTgUpdate(update tg.Update) {
	defer a.waitGroup.Done()
	defer a.DebugLogger.Println("⏹️ Closing TG update handler", update.UpdateID)

	var chatID types.TgChatID

	if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else if update.EditedMessage != nil {
		chatID = update.EditedMessage.Chat.ID
	} else {
		a.ErrorLogger.Printf("Unknown update #%d in chat (not Message nor EditedMessage nor CallbackQuery)", update.UpdateID)
		a.ErrorLogger.Println(update)
		return
	}

	ch, err := chat.New(chatID, a)
	if err != nil {
		a.ErrorLogger.Printf("Error initiating chat #%d: %v", chatID, err)
		a.TgAPI.Send(tg.NewMessage(chatID, internaErrorMsg))
		return
	}

	ch.HandleUpdate(update)
}

func (a *App) LoadState(chatID types.TgChatID) (state *store.State, err error) {
	// First, look in the persistent storage
	state, err = a.Store.LoadState(chatID)
	if err == nil {
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	// Second, look in the cache or create a new empty state
	iState, err := a.stateCache.Get(chatID)

	return iState.(*store.State), err
}

func (a *App) DropState(state *store.State) error {
	a.stateCache.Remove(state.ID)
	if state.IsAuthorized() {
		return a.Store.DeleteState(state.ID)
	}
	return nil
}

func (a *App) SaveState(state *store.State) error {
	if state.IsAuthorized() {
		a.stateCache.Remove(state.ID)
		return a.Store.SaveState(state)
	}

	return a.stateCache.Set(state.ID, state)
}

func (a *App) Send(m tg.Chattable) (tg.Message, error) {
	return a.TgAPI.Send(m)
}

func (a *App) RTSend(chatID types.TgChatID, cmd string, payload interface{}, reply interface{}) error {
	rt, ok := a.rtConns[chatID]
	if !ok {
		return fmt.Errorf("cannot find opened rt channel: %w", types.ErrNotFound)
	}
	replyBytes, err := rt.Send(cmd, payload)
	if err != nil {
		return fmt.Errorf("cannot send rt message: %w", err)
	}
	if reply != nil {
		if err := json.Unmarshal(replyBytes, reply); err != nil {
			return fmt.Errorf("cannot parse rt response: %w", err)
		}
	}
	return nil
}
