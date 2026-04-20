package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func multi(_ *resp.Array, conn *pubsub.Connection) {
	if conn.IsTransactionQueued {
		conn.Write(&resp.SimpleError{Val: []byte("ERR MULTI calls can not be nested")})
	}
	conn.IsTransactionQueued = true
	conn.TransactionCommands = make([]*resp.Array, 0)
	conn.Write(&resp.SimpleString{Val: []byte("OK")})
}
