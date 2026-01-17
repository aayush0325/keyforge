package commands

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// client handles the CLIENT command and its subcommands
func client(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'client' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	subCmd, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument for 'client' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	subCmdLower := strings.ToLower(string(subCmd.Str))

	switch subCmdLower {
	case "setinfo":
		// CLIENT SETINFO is used by clients to set metadata (lib-name, lib-ver)
		// We just acknowledge it - the data is informational only
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.W.Write(msg.ToBytes())
	case "setname":
		// CLIENT SETNAME sets the connection name
		if len(args.Val) >= 3 {
			name, ok := args.Val[2].(*resp.BulkString)
			if ok {
				conn.Name = string(name.Str)
			}
		}
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.W.Write(msg.ToBytes())
	case "getname":
		// CLIENT GETNAME returns the connection name
		if conn.Name == "" {
			conn.W.Write([]byte("$-1\r\n"))
		} else {
			res := resp.BulkString{Str: []byte(conn.Name), Size: len(conn.Name)}
			conn.W.Write(res.ToBytes())
		}
	default:
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.W.Write(msg.ToBytes())
	}
}
