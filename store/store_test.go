package store_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/FreeFeed/freefeed-tg-client/store"
	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/suite"
	"golang.org/x/text/language"
)

const maxSentRecords = 5

func TestExampleTestSuite(t *testing.T) { suite.Run(t, new(StoreTestSite)) }

type StoreTestSite struct {
	suite.Suite
	dir   string
	store store.Store
}

func (s *StoreTestSite) SetupTest() {
	var err error
	s.dir, err = os.MkdirTemp("", "test")
	s.NoError(err)
	s.store = store.NewFsStore(s.dir, store.FsMaxSentRecords(maxSentRecords))
	s.NotNil(s.store)
}

func (s *StoreTestSite) TearDownTest() {
	os.RemoveAll(s.dir)
}

// State

func (s *StoreTestSite) TestStateNotFound() {
	var chatID types.TgChatID = 123
	state, err := s.store.LoadState(chatID)
	s.ErrorIs(err, store.ErrNotFound)
	s.Nil(state)
}

func (s *StoreTestSite) TestStateStoreAndLoad() {
	var err error
	state := &store.State{ID: 123, ReactToMessageID: 321, Language: language.Russian}

	err = s.store.SaveState(state)
	s.NoError(err)

	state1, err := s.store.LoadState(state.ID)
	s.NoError(err)
	s.Equal(state, state1)
}

func (s *StoreTestSite) TestDeleteState() {
	var err error
	state := &store.State{ID: 123, ReactToMessageID: 321, Language: language.Russian}

	err = s.store.SaveState(state)
	s.NoError(err)

	err = s.store.DeleteState(state.ID)
	s.NoError(err)

	state1, err := s.store.LoadState(state.ID)
	s.ErrorIs(err, store.ErrNotFound)
	s.Nil(state1)
}

// List

func (s *StoreTestSite) TestListIDs() {
	var err error
	states := []*store.State{
		{ID: 123},
		{ID: 124},
		{ID: 125},
	}

	for _, state := range states {
		err = s.store.SaveState(state)
		s.NoError(err)
	}

	list, err := s.store.ListIDs()
	s.NoError(err)

	s.Len(list, len(states))
	for _, state := range states {
		s.Contains(list, state.ID)
	}

	// Removing one state
	err = s.store.DeleteState(states[0].ID)
	s.NoError(err)

	states = states[1:]

	list, err = s.store.ListIDs()
	s.NoError(err)

	s.Len(list, len(states))
	for _, state := range states {
		s.Contains(list, state.ID)
	}
}

// EventsQueue

func (s *StoreTestSite) TestEmptyEventsQueue() {
	queue, err := s.store.LoadAndDeleteQueue(123)
	s.NoError(err)
	s.Nil(queue)
}

func (s *StoreTestSite) TestEventsQueue() {
	const chatID = 123
	elements := []json.RawMessage{[]byte("1"), []byte("null"), []byte(`{"a": "b"}`)}

	for _, element := range elements {
		err := s.store.AddToQueue(chatID, element)
		s.NoError(err)
	}

	queue, err := s.store.LoadAndDeleteQueue(123)
	s.NoError(err)
	s.Len(queue, len(elements))

	// Should be empty
	queue, err = s.store.LoadAndDeleteQueue(123)
	s.NoError(err)
	s.Nil(queue)
}

// EventsStore

func (s *StoreTestSite) TestEmptySentMsgRecs() {
	rec, err := s.store.GetMsgRec(123, 1234)
	s.ErrorIs(err, store.ErrNotFound)
	s.Equal(rec, store.SentMsgRec{})
}

func (s *StoreTestSite) TestSentMsgRecs() {
	const chatID = 123
	recs := []store.SentMsgRec{{MessageID: 1234}, {MessageID: 1235}, {MessageID: 1236}}

	for _, rec := range recs {
		err := s.store.PutMsgRec(chatID, rec)
		s.NoError(err)
	}

	for _, rec := range recs {
		rec1, err := s.store.GetMsgRec(chatID, rec.MessageID)
		s.NoError(err)
		s.Equal(rec1, rec)
	}
}

func (s *StoreTestSite) TestMaxSentMsgRecs() {
	const chatID = 123
	recs := []store.SentMsgRec{
		{MessageID: 1234},
		{MessageID: 1235},
		{MessageID: 1236},
		{MessageID: 1237},
		{MessageID: 1238},
		{MessageID: 1239},
	}

	for _, rec := range recs {
		err := s.store.PutMsgRec(chatID, rec)
		s.NoError(err)
	}

	for i, rec := range recs {
		rec1, err := s.store.GetMsgRec(chatID, rec.MessageID)
		if i < len(recs)-maxSentRecords {
			s.ErrorIs(err, store.ErrNotFound)
			s.Equal(rec1, store.SentMsgRec{})
		} else {
			s.NoError(err)
			s.Equal(rec1, rec)
		}
	}
}

// Tracked posts

func (s *StoreTestSite) TestEmptyTrackedEntites() {
	const chatID = 123
	tracked, err := s.store.TrackedEntities(chatID)
	s.NoError(err)
	s.Equal(tracked, store.TrackedEntities{})
}

func (s *StoreTestSite) TestTrackedPosts() {
	const chatID = 123
	postID, _ := uuid.NewV4()
	postID2, _ := uuid.NewV4()

	ok, err := s.store.IsPostTracked(chatID, postID)
	s.NoError(err)
	s.False(ok)

	err = s.store.TrackPost(chatID, postID)
	s.NoError(err)

	ok, err = s.store.IsPostTracked(chatID, postID)
	s.NoError(err)
	s.True(ok)

	ok, err = s.store.IsPostTracked(chatID, postID2)
	s.NoError(err)
	s.False(ok)

	err = s.store.UntrackPost(chatID, postID)
	s.NoError(err)

	ok, err = s.store.IsPostTracked(chatID, postID)
	s.NoError(err)
	s.False(ok)
}
