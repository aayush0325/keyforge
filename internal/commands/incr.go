package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func incr(args *resp.Array, conn *pubsub.Connection) {
	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		if conn != nil {
			msg := resp.SimpleError{
				Val: []byte("wrong data type of list entry in 'incr' command")}
			conn.Write(&msg)
		}
		return
	}

	shardCh := db.GetShardChannel(string(key.Str))
	ch := make(chan []byte)
	cmd := db.NewCommandWithOptions(string(key.Str), nil, -1, ch, db.INCR, false)

	shardCh <- cmd

	res := <-ch

	if conn != nil {
		conn.Write(resp.RawMessage(res))
	}
}
