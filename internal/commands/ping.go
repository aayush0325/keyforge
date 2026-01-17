package commands

import (
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func ping(args *resp.Array, conn *pubsub.Connection) {
	if len(conn.Channels) > 0 {
		if len(args.Val) > 2 {
			msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'ping' command")}
			conn.W.Write(msg.ToBytes())
			return
		}

		pongMessage := &resp.BulkString{Str: []byte("pong"), Size: 4}

		var arg string
		if len(args.Val) == 1 {
			arg = ""
		} else {
			str, ok := args.Val[1].(*resp.BulkString)
			if !ok {
				msg := resp.SimpleError{Val: []byte("wrong data type of 2nd argument for 'ping' command")}
				conn.W.Write(msg.ToBytes())
				return
			}
			arg = string(str.Str)
		}
		responseArray := &resp.Array{Val: []resp.Message{pongMessage, &resp.BulkString{Str: []byte(arg), Size: len(arg)}}}
		conn.W.Write(responseArray.ToBytes())
		return
	}

	if len(args.Val) == 1 {
		msg := resp.SimpleString{Val: []byte("PONG")}
		conn.W.Write(msg.ToBytes())
		return
	}

	if len(args.Val) > 2 {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'ping' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	str, ok := args.Val[1].(*resp.BulkString)

	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 2nd argument for 'ping' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	conn.W.Write(str.ToBytes())
}
