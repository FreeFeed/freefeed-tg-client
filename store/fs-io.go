package store

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"

	"github.com/davidmz/mustbe"
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
	defer mustbe.CatchedAs(&outErr)

	cfg := new(optCfg)
	for _, opt := range options {
		opt(cfg)
	}

	lk, release := s.fileLock(chatID)
	defer release()
	lk.RLock()
	defer lk.RUnlock()

	filePath := path.Join(s.stateDirPath(chatID), baseName)

	data, err := ioutil.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		if cfg.MustExists {
			mustbe.Thrown(ErrNotFound)
		} else {
			return
		}
	} else {
		mustbe.Thrown(err)
	}

	mustbe.OK(json.Unmarshal(data, result))
	if cfg.DeleteFile {
		mustbe.OK(os.Remove(filePath))
	}

	return
}

func (s *fsStore) saveData(chatID tKey, baseName string, content interface{}) (err error) {
	defer mustbe.CatchedAs(&err)

	lk, release := s.fileLock(chatID)
	defer release()
	lk.Lock()
	defer lk.Unlock()

	data := mustbe.OKVal(json.Marshal(content)).([]byte)
	mustbe.OK(os.MkdirAll(s.stateDirPath(chatID), dirsPerm))

	filePath := path.Join(s.stateDirPath(chatID), baseName)
	mustbe.OK(ioutil.WriteFile(filePath, data, filesPerm))

	return
}

func (s *fsStore) updateData(chatID tKey, baseName string, result interface{}, processor func() error, options ...option) (outErr error) {
	defer mustbe.CatchedAs(&outErr)

	cfg := new(optCfg)
	for _, opt := range options {
		opt(cfg)
	}

	lk, release := s.fileLock(chatID)
	defer release()
	lk.Lock()
	defer lk.Unlock()

	filePath := path.Join(s.stateDirPath(chatID), baseName)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if cfg.MustExists || !errors.Is(err, os.ErrNotExist) {
			mustbe.Thrown(err)
		}
	} else {
		mustbe.OK(json.Unmarshal(data, result))
	}

	if mustbe.OKOr(processor(), errSkipUpdate) != nil {
		return
	}

	data = mustbe.OKVal(json.Marshal(result)).([]byte)

	mustbe.OK(ioutil.WriteFile(filePath, data, filesPerm))

	return
}
