package store

type FsOption func(s *fsStore)

func FsMaxSentRecords(n int) FsOption {
	return func(s *fsStore) { s.maxSentRecords = n }
}
