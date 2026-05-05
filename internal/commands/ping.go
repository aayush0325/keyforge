package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func ping(args *resp.Array, conn *pubsub.Connection) {
	if len(conn.Channels) > 0 {
		if len(args.Val) > 2 {
			msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'ping' command")}
			conn.Write(&msg)
			return
		}

		pongMessage := &resp.BulkString{Str: []byte("pong"), Size: 4}

		var arg string
		if len(args.Val) == 1 {
			arg = ""
		} else {
			str, ok := getBulkArg(args, 1, "ping", conn)
			if !ok {
				return
			}
			arg = string(str.Str)
		}
		responseArray := &resp.Array{Val: []resp.Message{pongMessage, &resp.BulkString{Str: []byte(arg), Size: len(arg)}}}
		conn.Write(responseArray)
		return
	}

	if len(args.Val) == 1 {
		msg := resp.SimpleString{Val: []byte("PONG")}
		conn.Write(&msg)
		return
	}

	if len(args.Val) > 2 {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'ping' command")}
		conn.Write(&msg)
		return
	}

	str, ok := getBulkArg(args, 1, "ping", conn)
	if !ok {
		return
	}

	conn.Write(str)
}
