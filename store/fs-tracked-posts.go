package store

import (
	"slices"

	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/gofrs/uuid"
)

type TrackedEntities struct {
	PostIDs []uuid.UUID
}

func (s *fsStore) TrackedEntities(chatID types.TgChatID) (TrackedEntities, error) {
	var tracked TrackedEntities
	err := s.loadData(chatID, trackedPostsFile, &tracked)
	if err != nil {
		return tracked, err
	}
	return tracked, nil
}

func (s *fsStore) TrackPost(chatID types.TgChatID, postID uuid.UUID) error {
	var tracked TrackedEntities
	return s.updateData(
		chatID,
		trackedPostsFile,
		&tracked,
		func() error {
			if !slices.Contains(tracked.PostIDs, postID) {
				tracked.PostIDs = append(tracked.PostIDs, postID)
			}
			return nil
		},
	)
}

func (s *fsStore) UntrackPost(chatID types.TgChatID, postID uuid.UUID) error {
	var tracked TrackedEntities
	return s.updateData(
		chatID,
		trackedPostsFile,
		&tracked,
		func() error {
			if idx := slices.Index(tracked.PostIDs, postID); idx >= 0 {
				slices.Delete(tracked.PostIDs, idx, idx+1)
			}
			return nil
		},
	)
}

func (s *fsStore) IsPostTracked(chatID types.TgChatID, postID uuid.UUID) (bool, error) {
	var tracked TrackedEntities
	if err := s.loadData(chatID, trackedPostsFile, &tracked); err != nil {
		return false, err
	}
	for _, id := range tracked.PostIDs {
		if id == postID {
			return true, nil
		}
	}
	return false, nil
}
