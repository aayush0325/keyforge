package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func llen(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'llen' command"),
		}
		conn.Write(&msg)
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type of list entry in 'lpush' command")}
		conn.Write(&msg)
		return
	}

	list := db.GetList(string(key.Str))
	if list == nil {
		msg := resp.Integer{Val: 0}
		conn.Write(&msg)
		return
	}

	msg := resp.Integer{Val: int64(list.Q.Len())}
	conn.Write(&msg)
}
