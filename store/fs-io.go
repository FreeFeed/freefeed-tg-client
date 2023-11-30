package store

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/davidmz/go-try"
)

var errSkipUpdate = errors.New("skip update")

type optCfg struct {
	MustExists bool
	DeleteFile bool
}

type option func(*optCfg)

func mustExists(cfg *optCfg) { cfg.MustExists = true }
func deleteFile(cfg *optCfg) { cfg.DeleteFile = true }

func (s *fsStore) loadData(chatID tKey, baseName string, result interface{}, options ...option) (outErr error) {
	defer try.HandleAs(&outErr)

	cfg := new(optCfg)
	for _, opt := range options {
		opt(cfg)
	}

	lk, release := s.fileLock(chatID)
	defer release()
	lk.RLock()
	defer lk.RUnlock()

	filePath := path.Join(s.stateDirPath(chatID), baseName)

	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		if cfg.MustExists {
			try.Throw(ErrNotFound)
		} else {
			return
		}
	} else {
		try.Throw(err)
	}

	try.It(json.Unmarshal(data, result))
	if cfg.DeleteFile {
		try.It(os.Remove(filePath))
	}

	return
}

func (s *fsStore) saveData(chatID tKey, baseName string, content interface{}) (err error) {
	defer try.HandleAs(&err)

	lk, release := s.fileLock(chatID)
	defer release()
	lk.Lock()
	defer lk.Unlock()

	data := try.ItVal(json.Marshal(content))
	try.It(os.MkdirAll(s.stateDirPath(chatID), dirsPerm))

	filePath := path.Join(s.stateDirPath(chatID), baseName)
	try.It(os.WriteFile(filePath, data, filesPerm))

	return
}

func (s *fsStore) updateData(chatID tKey, baseName string, result interface{}, processor func() error, options ...option) (outErr error) {
	defer try.HandleAs(&outErr)

	cfg := new(optCfg)
	for _, opt := range options {
		opt(cfg)
	}

	lk, release := s.fileLock(chatID)
	defer release()
	lk.Lock()
	defer lk.Unlock()

	filePath := path.Join(s.stateDirPath(chatID), baseName)

	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		if cfg.MustExists {
			try.Throw(ErrNotFound)
		} else {
			data = nil
		}
	} else {
		try.Throw(err)
	}

	if data != nil {
		try.It(json.Unmarshal(data, result))
	}

	if err = processor(); err == errSkipUpdate {
		return
	} else {
		try.Throw(err)
	}

	data = try.ItVal(json.Marshal(result))

	try.It(os.MkdirAll(s.stateDirPath(chatID), dirsPerm))
	try.It(os.WriteFile(filePath, data, filesPerm))

	return
}
