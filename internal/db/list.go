package db

import (
	"log"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/internal/ds"
)

type ListEntry struct {
	Mu sync.Mutex
	Q  ds.Deque[string]
	B  ds.Deque[chan struct{}]
}

type ListsMap struct {
	Mu sync.Mutex
	L  map[string]*ListEntry
}

var ListOnce sync.Once
var lists *ListsMap

// Get a list from the global store with a key, if it doesn't exist this function
// will return nil
func GetList(key string) *ListEntry {
	ListOnce.Do(func() {
		log.Printf("ListsMap: Initializing...This should happen only once")
		lists = &ListsMap{L: make(map[string]*ListEntry)}
	})

	lists.Mu.Lock()
	defer lists.Mu.Unlock()

	val, ok := lists.L[key]
	if !ok {
		return nil
	}
	return val
}

// Get a list from the global store with a key, if it doesn't exist this function
// will create one and return it
func CreateOrGetList(key string) *ListEntry {
	ListOnce.Do(func() {
		log.Printf("ListsMap: Initializing...This should happen only once")
		lists = &ListsMap{L: make(map[string]*ListEntry)}
	})

	lists.Mu.Lock()
	defer lists.Mu.Unlock()

	val, ok := lists.L[key]
	if !ok {
		lists.L[key] = &ListEntry{Q: *ds.NewDeque[string](), B: *ds.NewDeque[chan struct{}]()}
		return lists.L[key]
	}
	return val
}

func DeleteList(key string) {
	lists.Mu.Lock()
	defer lists.Mu.Unlock()

	_, ok := lists.L[key]
	if !ok {
		log.Printf("ListsMap: Trying to delete list :%s, but it doesn't exist", key)
		return
	}
	log.Printf("ListsMap: Deleting list :%s", key)
	delete(lists.L, key)
}
