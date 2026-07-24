package db

import "log"

type DBEventType string

const (
	DBCreate DBEventType = "create"
	DBRead   DBEventType = "read"
	DBUpdate DBEventType = "update"
	DBDelete DBEventType = "delete"
)

type DBEvent struct {
	Table string
	Type  DBEventType
	Data  any
}

func (db *DB) Subscribe(bufferSize int) (int64, <-chan DBEvent) {
	db.Lock()
	defer db.Unlock()

	id := db.nextID
	db.nextID++

	channel := make(chan DBEvent, bufferSize)
	db.subscribers[id] = channel

	return id, channel
}

func (db *DB) Unsubscribe(id int64) {
	db.Lock()
	defer db.Unlock()

	if channel, ok := db.subscribers[id]; ok {
		close(channel)
		delete(db.subscribers, id)
	}
}

func (db *DB) Emit(dbEvent DBEvent) {
	db.RLock()
	defer db.RUnlock()

	for id, channel := range db.subscribers {
		select {
		case channel <- dbEvent:
		default:
			log.Printf("[db] subscriber %d slow; event dropped", id)
		}
	}
}
