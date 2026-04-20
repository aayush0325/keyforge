package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func discard(_ *resp.Array, conn *pubsub.Connection) {
	if !conn.IsTransactionQueued {
		conn.Write(resp.RawMessage("-ERR DISCARD without MULTI\r\n"))
		conn.IsTransactionQueued = false
		conn.IsTransactionRunning = false
		conn.TransactionCommands = nil
		conn.TransactionResponse = nil
		return
	}

	conn.W.Write(resp.RawMessage("+OK\r\n"))
	conn.IsTransactionQueued = false
	conn.IsTransactionRunning = false
	conn.TransactionCommands = nil
	conn.TransactionResponse = nil
}
