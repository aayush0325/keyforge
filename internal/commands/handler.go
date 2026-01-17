package commands

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// DebugMode enables logging of all commands when set to true
var DebugMode bool

func logCommand(arr *resp.Array) {
	if !DebugMode {
		return
	}
	var parts []string
	for _, item := range arr.Val {
		if bs, ok := item.(*resp.BulkString); ok {
			parts = append(parts, string(bs.Str))
		}
	}
	log.Printf("[DEBUG] %s", strings.Join(parts, " "))
}

var allowedInSubscribedMode = map[string]struct{}{
	"subscribe":    {},
	"unsubscribe":  {},
	"psubscribe":   {},
	"punsubscribe": {},
	"ping":         {},
	"quit":         {},
	"reset":        {},
}

func ExecuteCommands(msg resp.Message, conn *pubsub.Connection) {
	arr, ok := msg.(*resp.Array)
	if !ok {
		return // Commands are sent via an array of bulk strings
	}

	cmd, ok := arr.Val[0].(*resp.BulkString)
	if !ok {
		return // Commands are sent via an array of bulk strings
	}

	logCommand(arr)

	cmdLower := string(bytes.ToLower(cmd.Str))

	// Check if client is in subscribed mode
	if len(conn.Channels) > 0 {
		if _, allowed := allowedInSubscribedMode[cmdLower]; !allowed {
			errMsg := fmt.Sprintf(
				"ERR Can't execute '%s': only (P|S)SUBSCRIBE / (P|S)UNSUBSCRIBE / PING / QUIT / RESET are allowed in this context",
				cmdLower)
			err := resp.SimpleError{Val: []byte(errMsg)}
			conn.W.Write(err.ToBytes())
			return
		}
	}

	switch cmdLower {
	case "echo":
		echo(arr, conn)
	case "ping":
		ping(arr, conn)
	case "hello":
		hello(arr, conn)
	case "client":
		client(arr, conn)
	case "command":
		command(arr, conn)
	case "set":
		set(arr, conn)
	case "setnx":
		setnx(arr, conn)
	case "get":
		get(arr, conn)
	case "del":
		del(arr, conn)
	case "exists":
		exists(arr, conn)
	case "rpush":
		rpush(arr, conn)
	case "lpush":
		lpush(arr, conn)
	case "llen":
		llen(arr, conn)
	case "lrange":
		lrange(arr, conn)
	case "lpop":
		lpop(arr, conn)
	case "blpop":
		blpop(arr, conn)
	case "config":
		config(arr, conn)
	case "type":
		typeCommand(arr, conn)
	case "subscribe":
		subscribe(arr, conn)
	case "publish":
		publish(arr, conn)
	case "unsubscribe":
		unsubscribe(arr, conn)
	case "xadd":
		xadd(arr, conn)
	case "xrange":
		xrange(arr, conn)
	case "xread":
		xread(arr, conn)
	default:
		commandDoesntExist(arr, conn)
	}
}
