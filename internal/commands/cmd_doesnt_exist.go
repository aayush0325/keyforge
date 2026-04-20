package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func commandDoesntExist(_ *resp.Array, conn *pubsub.Connection) {
	err := resp.SimpleError{Val: ([]byte)("This command doesn't exist in the server")}
	conn.Write(&err)
}
