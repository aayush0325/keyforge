package commands

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func exists(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 2, "exists", conn) {
		return
	}

	existsCount := int64(0)

	// Handle multiple keys
	for i := 1; i < len(args.Val); i++ {
		key, ok := getBulkArgMsg(args, i, conn, "wrong data type for argument of 'exists' command")
		if !ok {
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
	conn.Write(&res)
}
