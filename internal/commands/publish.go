package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func publish(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 3, "publish", conn) {
		return
	}

	channel, ok := getBulkArg(args, 1, "publish", conn)
	if !ok {
		return
	}

	message, ok := getBulkArg(args, 2, "publish", conn)
	if !ok {
		return
	}

	payload := resp.Array{
		Val: []resp.Message{
			&resp.BulkString{Str: []byte("message"), Size: 7},
			channel,
			message,
		},
	}

	cons := pubsub.Instance.GetMap(string(channel.Str))

	count := int64(len(cons))

	go pubsub.Instance.DeliverMessage(cons, payload.ToBytes())

	res := resp.Integer{Val: count}
	conn.Write(&res)
}
