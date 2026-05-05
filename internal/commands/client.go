package commands

import (
	"strings"

	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

// client handles the CLIENT command and its subcommands
func client(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 2, "client", conn) {
		return
	}

	subCmd, ok := getBulkArg(args, 1, "client", conn)
	if !ok {
		return
	}

	subCmdLower := strings.ToLower(string(subCmd.Str))

	switch subCmdLower {
	case "setinfo":
		// CLIENT SETINFO is used by clients to set metadata (lib-name, lib-ver)
		// We just acknowledge it - the data is informational only
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.Write(&msg)
	case "setname":
		// CLIENT SETNAME sets the connection name
		if len(args.Val) >= 3 {
			name, ok := args.Val[2].(*resp.BulkString)
			if ok {
				conn.Name = string(name.Str)
			}
		}
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.Write(&msg)
	case "getname":
		// CLIENT GETNAME returns the connection name
		if conn.Name == "" {
			conn.Write(&resp.BulkString{Str: nil, Size: -1})
		} else {
			res := resp.BulkString{Str: []byte(conn.Name), Size: len(conn.Name)}
			conn.Write(&res)
		}
	default:
		msg := resp.SimpleString{Val: []byte("OK")}
		conn.Write(&msg)
	}
}
