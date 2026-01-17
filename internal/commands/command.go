package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func command(_ *resp.Array, conn *pubsub.Connection) {
	// kept for redis-cli compatablity
	// TODO: implement docs for redis-cli using this
	conn.W.Write([]byte("*0\r\n"))
}
