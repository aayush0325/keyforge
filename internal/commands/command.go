package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func command(_ *resp.Array, conn *pubsub.Connection) {
	// kept for redis-cli compatablity
	// TODO: implement docs for redis-cli using this
	conn.Write(&resp.Array{Val: []resp.Message{}})
}
