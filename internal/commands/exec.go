package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/rdb"
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

	// Hold the dataset lock for the whole transaction so that a dump can neither
	// observe half of it nor cause a write to be both dumped and propagated.
	locked := false
	if hasWriteCommand(conn.TransactionCommands) {
		rdb.LockDataset()
		locked = true
	}

	for _, cmd := range conn.TransactionCommands {
		ExecuteCommands(cmd, conn)
	}

	if locked {
		rdb.UnlockDataset()
	}

	conn.W.Write(conn.TransactionResponse.ToBytes())
	conn.IsTransactionQueued = false
	conn.IsTransactionRunning = false
	conn.TransactionCommands = nil
	conn.TransactionResponse = nil
}
