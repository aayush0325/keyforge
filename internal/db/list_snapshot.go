package db

import (
	"log"

	"github.com/aayush0325/keyforge/internal/ds"
)

// initLists lazily creates the global list map, every entry point of the list
// store has to go through it.
func initLists() {
	ListOnce.Do(func() {
		log.Printf("ListsMap: Initializing...This should happen only once")
		lists = &ListsMap{L: make(map[string]*ListEntry)}
	})
}

// SnapshotLists returns a copy of every list, entries are in list order. Used by
// the RDB dump path (persistence and FULLRESYNC).
func SnapshotLists() map[string][]string {
	initLists()

	lists.Mu.Lock()
	defer lists.Mu.Unlock()

	snapshot := make(map[string][]string, len(lists.L))

	for key, entry := range lists.L {
		entry.Mu.Lock()
		if entry.Q.Len() > 0 {
			elements := make([]string, entry.Q.Len())
			copy(elements, entry.Q.Buf)
			snapshot[key] = elements
		}
		entry.Mu.Unlock()
	}

	return snapshot
}

// ResetList replaces the content of a list with the given elements. It is used
// while loading an RDB payload. Registered blocking consumers are kept, so a
// BLPOP waiting on the list still gets woken up by the next push.
func ResetList(key string, elements []string) {
	initLists()

	list := CreateOrGetList(key)

	list.Mu.Lock()
	defer list.Mu.Unlock()

	list.Q = *ds.NewDeque[string]()
	for _, element := range elements {
		list.Q.PushBack(element)
	}
}

// FlushLists drops every list, used before loading an RDB payload.
func FlushLists() {
	initLists()

	lists.Mu.Lock()
	defer lists.Mu.Unlock()

	lists.L = make(map[string]*ListEntry)
}
