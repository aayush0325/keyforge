package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func echo(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'echo' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	str, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type argument for 'echo' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	conn.W.Write(str.ToBytes())
}
