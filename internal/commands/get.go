package commands

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func get(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 2, "get", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "get", conn)
	if !ok {
		return
	}

	keyStr := string(key.Str)
	channel := make(chan []byte, 1)
	cmd := db.NewCommand(keyStr, nil, -1, channel, db.GET)

	// Route to the appropriate shard based on key
	shardCh := db.GetShardChannel(keyStr)
	shardCh <- cmd

	value, ok := <-channel
	if ok {
		conn.Write(resp.RawMessage(value))
		close(channel)
		return
	}
}
