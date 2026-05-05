package commands

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func llen(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 2, "llen", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "llen", conn)
	if !ok {
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
