package commands

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/rdb"
	"github.com/aayush0325/keyforge/internal/resp"
)

// DebugMode enables logging of all commands when set to true
var DebugMode bool

// writeCommands are the commands that modify the dataset, they are the ones
// streamed to the replicas after a full resynchronization.
var writeCommands = map[string]struct{}{
	"set":   {},
	"setnx": {},
	"del":   {},
	"incr":  {},
	"lpush": {},
	"rpush": {},
	"lpop":  {},
	"xadd":  {},
}

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

func isWriteCommand(cmd string) bool {
	_, ok := writeCommands[cmd]
	return ok
}

// hasWriteCommand reports whether any of the queued commands modifies the
// dataset.
func hasWriteCommand(cmds []*resp.Array) bool {
	for _, cmd := range cmds {
		if len(cmd.Val) == 0 {
			continue
		}
		name, ok := cmd.Val[0].(*resp.BulkString)
		if !ok {
			continue
		}
		if isWriteCommand(string(bytes.ToLower(name.Str))) {
			return true
		}
	}
	return false
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
			conn.Write(&err)
			return
		}
	}

	if conn.IsTransactionQueued && (cmdLower != "exec" && cmdLower != "discard") {
		conn.Write(&resp.SimpleString{Val: []byte("QUEUED")})
		conn.TransactionCommands = append(conn.TransactionCommands, arr)
		return
	}

	// Write commands are executed while holding the dataset lock and hold it
	// until they were propagated to the replicas. A dump (SAVE, BGSAVE or the
	// FULLRESYNC of a new replica) therefore either sees the write or receives
	// it on the replication stream, never both. Replicas never take the lock:
	// they apply whatever the master sends them.
	//
	// Commands of a transaction are covered by the lock exec() takes around the
	// whole transaction, taking it again here would deadlock.
	writeCmd := isWriteCommand(cmdLower) && !conf.IsReplica
	locked := false
	if writeCmd && !conn.IsTransactionRunning {
		rdb.LockDataset()
		locked = true
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
	case "incr":
		incr(arr, conn)
	case "multi":
		multi(arr, conn)
	case "exec":
		exec(arr, conn)
	case "discard":
		discard(arr, conn)
	case "info":
		info(arr, conn)
	case "replconf":
		replconf(arr, conn)
	case "psync":
		psync(arr, conn)
	case "wait":
		wait(arr, conn)
	case "save":
		saveCommand(arr, conn)
	case "bgsave":
		bgsave(arr, conn)
	case "lastsave":
		lastsaveCommand(arr, conn)
	case "debug":
		debug(arr, conn)
	default:
		commandDoesntExist(arr, conn)
	}

	if conf.IsReplica && conn.IsMaster {
		// Everything the master streams to us moves our offset forward
		conf.AddOffset(uint64(len(arr.ToBytes())))
	}

	if writeCmd {
		pubsub.Instance.PropagateToReplicas(arr.ToBytes())
		conf.AddOffset(uint64(len(arr.ToBytes())))
	}

	if locked {
		rdb.UnlockDataset()
	}

	log.Printf("offset: %d", conf.GetOffset())
}
