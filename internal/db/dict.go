package db

import (
	"log"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

type Shard struct {
	kv map[string]Entry
	ch chan Command
}

func NewCommand(key string, value []byte, ttl int64, c chan []byte, op MapCommands) Command {
	return Command{key: key, value: value, ttl: ttl, c: c, operation: op}
}

func NewCommandWithOptions(key string, value []byte, ttl int64, c chan []byte, op MapCommands, nx bool) Command {
	return Command{key: key, value: value, ttl: ttl, c: c, operation: op, nx: nx}
}

// a GET, SET, or TYPE command will be passed into the channel as this struct
type Command struct {
	key   string      // key as a string
	value []byte      // value as a byte array
	ttl   int64       // ttl as a 64 bit signed integer, (negative ttl = infinite)
	c     chan []byte // channel where we expect the goroutine to push the
	// return value
	operation MapCommands // type of command being pushed
	nx        bool        // NX flag: only set if key does not exist
}

type MapCommands int

// Enum defining types of commands we can push into the recieving channels of a shard
const (
	GET MapCommands = iota // iota starts at 0
	SET
	TYPE
	CLEANUP
	DEL
	EXISTS
)

var (
	shards [Shards]*Shard // Array of shards, index is determined by the hash value
	KVOnce sync.Once
)

func InitKVStore() {
	KVOnce.Do(func() {
		log.Printf("KVStore: Initializing %d shards...This should happen only once", Shards)
		for i := range Shards {
			shards[i] = &Shard{
				kv: make(map[string]Entry),
				ch: make(chan Command, 4096), // buffered channel
			}
			go shardLoop(shards[i]) // for each shard launch a goroutine which acts as the single thread interacting with that shard, hence we don't lock
			go deleteExpiredKeysForShard(shards[i])
		}
	})
}

// GetShardChannel returns the command channel for the shard that should handle the given key
func GetShardChannel(key string) chan Command {
	idx := shardForKey(key)
	return shards[idx].ch
}

func shardLoop(s *Shard) {
	for cmd := range s.ch {
		// Don't use "default" here as it consumes unnecessary cpu: https://stackoverflow.com/questions/55367231/golang-for-select-loop-consumes-100-of-cpu
		switch cmd.operation {
		case GET:
			handleGetCommand(s, cmd)
		case SET:
			handleSetCommand(s, cmd)
		case TYPE:
			handleTypeCommand(s, cmd)
		case CLEANUP:
			handleCleanupCommand(s)
		case DEL:
			handleDelCommand(s, cmd)
		case EXISTS:
			handleExistsCommand(s, cmd)
		}
	}
}

// delete expired keys in a shard
func handleCleanupCommand(s *Shard) {
	now := time.Now()
	for key, val := range s.kv {
		if now.After(val.ExpiresAt) && !val.ExpiresAt.IsZero() {
			delete(s.kv, key)
		}
	}
}

func deleteExpiredKeysForShard(s *Shard) {
	deleteOperationTicker := time.NewTicker(300 * time.Second)
	defer deleteOperationTicker.Stop()
	for {
		<-deleteOperationTicker.C
		s.ch <- NewCommand("", nil, 0, nil, CLEANUP) // push a cleanup command into the shard channel
		// this is done so that we don't run into a race condition where this function and the shard
		// channel access the map at the same time
	}
}

// When we get a GET command
func handleGetCommand(s *Shard, g Command) {
	val, ok := s.kv[g.key]
	if !ok {
		g.c <- ([]byte("$-1\r\n")) // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	if time.Now().After(val.ExpiresAt) && !val.ExpiresAt.IsZero() {
		delete(s.kv, g.key)
		g.c <- ([]byte("$-1\r\n")) // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	msg := resp.BulkString{
		Str:  []byte(val.Value),
		Size: len(val.Value),
	}

	g.c <- msg.ToBytes()
}

// ttl > 0 -> positive ttl in miliseconds
// ttl = 0 -> no operation
// ttl < 0 -> value stays in map unless explicitly deleted
func handleSetCommand(shard *Shard, cmd Command) {
	if cmd.ttl == 0 {
		cmd.c <- []byte("+OK\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	// NX: only set if key does not exist
	if cmd.nx {
		existing, exists := shard.kv[cmd.key]
		// Check if key exists and is not expired
		if exists && (existing.ExpiresAt.IsZero() || time.Now().Before(existing.ExpiresAt)) {
			cmd.c <- []byte("$-1\r\n") // return nil if key already exists
			return
		}
	}

	if cmd.ttl < 0 {
		shard.kv[cmd.key] = Entry{Value: cmd.value}
		cmd.c <- []byte("+OK\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	expiry := time.Now().Add(time.Millisecond * time.Duration(cmd.ttl))

	shard.kv[cmd.key] = Entry{Value: cmd.value, ExpiresAt: expiry}
	cmd.c <- []byte("+OK\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
}

func handleTypeCommand(s *Shard, cmd Command) {
	val, ok := s.kv[cmd.key]
	if !ok {
		cmd.c <- []byte("+none\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	if time.Now().After(val.ExpiresAt) && !val.ExpiresAt.IsZero() {
		delete(s.kv, cmd.key)
		cmd.c <- []byte("+none\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
		return
	}

	cmd.c <- []byte("+string\r\n") // use raw byte arrays where we can to reduce conversion cost by CPU
}

// handleDelCommand deletes a key and returns 1 if it existed, 0 otherwise
func handleDelCommand(s *Shard, cmd Command) {
	val, ok := s.kv[cmd.key]
	if !ok {
		cmd.c <- []byte(":0\r\n") // key did not exist
		return
	}

	// Check if key is expired
	if !val.ExpiresAt.IsZero() && time.Now().After(val.ExpiresAt) {
		delete(s.kv, cmd.key)
		cmd.c <- []byte(":0\r\n") // key was expired, treat as not existing
		return
	}

	delete(s.kv, cmd.key)
	cmd.c <- []byte(":1\r\n") // key existed and was deleted
}

// handleExistsCommand checks if a key exists and returns 1 if it does, 0 otherwise
func handleExistsCommand(s *Shard, cmd Command) {
	val, ok := s.kv[cmd.key]
	if !ok {
		cmd.c <- []byte(":0\r\n") // key does not exist
		return
	}

	// Check if key is expired
	if !val.ExpiresAt.IsZero() && time.Now().After(val.ExpiresAt) {
		delete(s.kv, cmd.key)
		cmd.c <- []byte(":0\r\n") // key was expired, treat as not existing
		return
	}

	cmd.c <- []byte(":1\r\n") // key exists
}
