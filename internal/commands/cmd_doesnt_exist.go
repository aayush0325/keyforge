package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func commandDoesntExist(_ *resp.Array, conn *pubsub.Connection) {
	err := resp.SimpleError{Val: ([]byte)("This command doesn't exist in the server")}
	conn.W.Write(err.ToBytes())
}
