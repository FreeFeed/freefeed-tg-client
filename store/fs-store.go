package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/davidmz/freefeed-tg-client/types"
)

type tKey = types.TgChatID

// NewFsStore creates a new file-based Store.
func NewFsStore(dirName string) Store {
	return &fsStore{
		dirName:   dirName,
		fileLocks: make(map[tKey]*sync.RWMutex),
	}
}

const (
	dirsPerm  = 0775
	filesPerm = 0600

	stateFile        = "state.json"
	queueFile        = "queue.json"
	sentEventsFile   = "sent-events.json"
	trackedPostsFile = "tracked-posts.json"
)

type fsStore struct {
	dirLock   sync.RWMutex
	fileLocks map[tKey]*sync.RWMutex
	dirName   string
}

func (s *fsStore) fileLock(key tKey) (*sync.RWMutex, func()) {
	s.dirLock.RLock()
	lk, ok := s.fileLocks[key]
	if ok {
		return lk, s.dirLock.RUnlock
	}
	s.dirLock.RUnlock()

	s.dirLock.Lock()
	if lk, ok = s.fileLocks[key]; !ok {
		lk = new(sync.RWMutex)
		s.fileLocks[key] = lk
	}

	return lk, s.dirLock.Unlock
}

func (s *fsStore) stateDirPath(chatID tKey) string {
	return path.Join(s.dirName, strconv.FormatInt(chatID, 10))
}

func (s *fsStore) LoadState(chatID tKey) (*State, error) {
	state := &State{}
	err := s.loadData(chatID, stateFile, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *fsStore) SaveState(state *State) error {
	return s.saveData(state.ID, stateFile, state)
}

func (s *fsStore) DeleteState(chatID tKey) error {
	s.dirLock.Lock()
	defer s.dirLock.Unlock()

	if err := os.RemoveAll(s.stateDirPath(chatID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove state files: %w", err)
	}

	delete(s.fileLocks, chatID)
	return nil
}

func (s *fsStore) ListIDs() ([]tKey, error) {
	s.dirLock.RLock()
	defer s.dirLock.RUnlock()

	entries, err := os.ReadDir(s.dirName)
	if err != nil {
		return nil, err
	}

	var ids []tKey
	for _, ent := range entries {
		if ent.IsDir() {
			if n, err := strconv.ParseInt(ent.Name(), 10, 64); err == nil {
				ids = append(ids, n)
			}
			// Or, if we cannot parse dir name, ignore this directory
		}
	}

	return ids, nil
}

func (s *fsStore) AddToQueue(chatID tKey, entry json.RawMessage) error {
	var queue []json.RawMessage
	return s.updateData(chatID, queueFile, &queue, func() error {
		queue = append(queue, entry)
		return nil
	})
}

func (s *fsStore) LoadAndDeleteQueue(chatID tKey) ([]json.RawMessage, error) {
	var queue []json.RawMessage
	if err := s.loadData(chatID, queueFile, &queue, deleteFile); err != nil {
		return nil, err
	}
	return queue, nil
}
