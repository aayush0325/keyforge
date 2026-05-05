package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func echo(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 2, "echo", conn) {
		return
	}

	str, ok := getBulkArg(args, 1, "echo", conn)
	if !ok {
		return
	}

	conn.Write(str)
}
