package domain

type WatcherEvent struct {
	Path string
	Type DocActionType
}
