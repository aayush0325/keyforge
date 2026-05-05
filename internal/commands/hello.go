package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

// hello handles the HELLO command
// Since this server only supports RESP2, we return NOPROTO error for RESP3 requests
func hello(args *resp.Array, conn *pubsub.Connection) {
	// HELLO with no args or HELLO 2 is valid for RESP2
	if len(args.Val) == 1 {
		// No version specified, return server info as array
		sendHelloResponse(conn)
		return
	}

	version, ok := getBulkArgMsg(args, 1, conn, "ERR Protocol version is not an integer or out of range")
	if !ok {
		return
	}

	versionStr := string(version.Str)
	if versionStr == "2" {
		sendHelloResponse(conn)
		return
	}

	// For RESP3 (version 3) or higher, return NOPROTO error
	msg := resp.SimpleError{Val: []byte("NOPROTO sorry this Redis does not support RESP3")}
	conn.Write(&msg)
}

func sendHelloResponse(conn *pubsub.Connection) {
	// Return minimal server info as an array (RESP2 style)
	// Real Redis returns a map, but RESP2 uses arrays
	response := resp.Array{
		Val: []resp.Message{
			&resp.BulkString{Str: []byte("server"), Size: 6},
			&resp.BulkString{Str: []byte("redis"), Size: 5},
			&resp.BulkString{Str: []byte("version"), Size: 7},
			&resp.BulkString{Str: []byte("7.0.0"), Size: 5},
			&resp.BulkString{Str: []byte("proto"), Size: 5},
			&resp.Integer{Val: 2},
		},
	}
	conn.Write(&response)
}
