package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func exists(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'exists' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	existsCount := int64(0)

	// Handle multiple keys
	for i := 1; i < len(args.Val); i++ {
		key, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{
				Val: []byte("wrong data type for argument of 'exists' command"),
			}
			conn.W.Write(msg.ToBytes())
			return
		}

		keyStr := string(key.Str)

		// Check if key exists as a list first
		if db.GetList(keyStr) != nil {
			existsCount++
			continue
		}

		// Check in KV store
		channel := make(chan []byte, 1)
		cmd := db.NewCommand(keyStr, nil, 0, channel, db.EXISTS)

		shardCh := db.GetShardChannel(keyStr)
		shardCh <- cmd

		value, ok := <-channel
		if ok {
			// Check if the response indicates key exists (":1\r\n")
			if len(value) >= 4 && value[0] == ':' && value[1] == '1' {
				existsCount++
			}
			close(channel)
		}
	}

	res := resp.Integer{Val: existsCount}
	conn.W.Write(res.ToBytes())
}
