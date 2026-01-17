package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/streams"
)

func typeCommand(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'type' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 2nd argument of 'type' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	keyStr := string(key.Str)

	// Check if key exists as a list first
	if db.GetList(keyStr) != nil {
		conn.W.Write([]byte("+list\r\n"))
		return
	}

	// Check if key exists as a stream
	streams.Global.Mu.Lock()
	_, ok = streams.Global.KV[keyStr]
	streams.Global.Mu.Unlock()
	if ok {
		conn.W.Write([]byte("+stream\r\n"))
		return
	}

	// Check in KV store
	channel := make(chan []byte, 1)
	cmd := db.NewCommand(keyStr, nil, -1, channel, db.TYPE)

	shardCh := db.GetShardChannel(keyStr)
	shardCh <- cmd

	value, ok := <-channel
	if ok {
		conn.W.Write(value)
		close(channel)
		return
	}
}
