package commands

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func incr(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 2, "incr", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "incr", conn)
	if !ok {
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
