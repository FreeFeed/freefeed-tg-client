package store

import (
	"fmt"

	"github.com/FreeFeed/freefeed-tg-client/frf"
	"github.com/FreeFeed/freefeed-tg-client/types"
)

type SentMsgRec struct {
	MessageID int
	Event     *frf.Event
	// .ReplyToMessage.MessageID
	ReplyToID int
}

func (s *fsStore) GetMsgRec(chatID types.TgChatID, messageID int) (SentMsgRec, error) {
	var records []SentMsgRec

	if err := s.loadData(chatID, sentEventsFile, &records); err != nil {
		return SentMsgRec{}, err
	}

	for _, record := range records {
		if record.MessageID == messageID {
			return record, nil
		}
	}
	return SentMsgRec{}, fmt.Errorf("cannot find event data for this message: %w", ErrNotFound)
}

func (s *fsStore) PutMsgRec(chatID types.TgChatID, rec SentMsgRec) error {
	var records []SentMsgRec
	return s.updateData(chatID, sentEventsFile, &records, func() error {
		records = append(records, rec)
		if len(records) > s.maxSentRecords {
			records = records[len(records)-s.maxSentRecords:]
		}
		return nil
	})
}
