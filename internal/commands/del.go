package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func del(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'del' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	deletedCount := int64(0)

	// Handle multiple keys
	for i := 1; i < len(args.Val); i++ {
		key, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{
				Val: []byte("wrong data type for argument of 'del' command"),
			}
			conn.W.Write(msg.ToBytes())
			return
		}

		keyStr := string(key.Str)
		channel := make(chan []byte, 1)
		cmd := db.NewCommand(keyStr, nil, 0, channel, db.DEL)

		// Route to the appropriate shard based on key
		shardCh := db.GetShardChannel(keyStr)
		shardCh <- cmd

		value, ok := <-channel
		if ok {
			// Check if the response indicates a successful delete (":1\r\n")
			if len(value) >= 4 && value[0] == ':' && value[1] == '1' {
				deletedCount++
			}
			close(channel)
		}
	}

	res := resp.Integer{Val: deletedCount}
	conn.W.Write(res.ToBytes())
}
