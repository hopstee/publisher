package domain

type WatcherEventType string

const (
	UpsertWatcherEventType WatcherEventType = "UPSERT"
	DeleteWatcherEventType WatcherEventType = "DELETE"
)

type WatcherEvent struct {
	Path string
	Type WatcherEventType
}
