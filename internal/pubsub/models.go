package pubsub

import (
	"bufio"
	"sync"
)

// The global pub sub instance is represented by this struct which contains a mapping of each client to
// an array of channels it is subscribed to + channels mapped to an array of connections
type Global struct {
	ChannelToClient map[string]map[*Connection]struct{}
	Mu              sync.RWMutex
}

// A "connection" with a client is represented as this struct, this is done to
// keep track of subcribed/unsubscribed modes and number of subscribed channels
type Connection struct {
	W        *bufio.Writer
	Channels map[string]struct{}
	Name     string // connection name set by CLIENT SETNAME
	Mu       sync.Mutex // protects W for concurrent writes
}

var PubSubOnce sync.Once
var Instance Global

func InitPubSub() {
	PubSubOnce.Do(func() {
		Instance = Global{
			ChannelToClient: make(map[string]map[*Connection]struct{}),
		}
	})
}
