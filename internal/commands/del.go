package commands

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func del(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 2, "del", conn) {
		return
	}

	deletedCount := int64(0)

	for i := 1; i < len(args.Val); i++ {
		key, ok := getBulkArgMsg(args, i, conn, "wrong data type for argument of 'del' command")
		if !ok {
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

	if conn != nil {
		res := resp.Integer{Val: deletedCount}
		conn.Write(&res)
	}
}
