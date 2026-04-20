package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func exec(_ *resp.Array, conn *pubsub.Connection) {
	if !conn.IsTransactionQueued {
		conn.Write(resp.RawMessage("-ERR EXEC without MULTI\r\n"))
		conn.IsTransactionQueued = false
		conn.IsTransactionRunning = false
		conn.TransactionCommands = nil
		conn.TransactionResponse = nil
		return
	}

	if len(conn.TransactionCommands) == 0 {
		conn.Write(&resp.Array{Val: []resp.Message{}})
		conn.IsTransactionQueued = false
		conn.IsTransactionRunning = false
		conn.TransactionCommands = nil
		conn.TransactionResponse = nil
		return
	}

	conn.IsTransactionQueued = false
	conn.IsTransactionRunning = true
	conn.TransactionResponse = &resp.Array{}

	for _, cmd := range conn.TransactionCommands {
		ExecuteCommands(cmd, conn)
	}

	conn.W.Write(conn.TransactionResponse.ToBytes())
	conn.IsTransactionQueued = false
	conn.IsTransactionRunning = false
	conn.TransactionCommands = nil
	conn.TransactionResponse = nil
}
