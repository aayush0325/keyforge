package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func publish(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 3 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'publish' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	channel, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type argument for 'publish' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	message, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		errormessage := resp.SimpleError{
			Val: []byte("wrong data type argument for 'publish' command"),
		}
		conn.W.Write(errormessage.ToBytes())
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
	conn.W.Write(res.ToBytes())
}
