package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func get(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'get' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 2nd argument of 'get' command"),
		}
		conn.W.Write(msg.ToBytes())
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
		conn.W.Write(value)
		close(channel)
		return
	}
}
