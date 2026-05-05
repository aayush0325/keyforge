package commands

import (
	"fmt"
	"strconv"

	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func requireExactArgs(args *resp.Array, n int, cmd string, conn *pubsub.Connection) bool {
	if len(args.Val) != n {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for '" + cmd + "' command")}
		conn.Write(&msg)
		return false
	}
	return true
}

func requireMinArgs(args *resp.Array, n int, cmd string, conn *pubsub.Connection) bool {
	if len(args.Val) < n {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for '" + cmd + "' command")}
		conn.Write(&msg)
		return false
	}
	return true
}

func getBulkArg(args *resp.Array, idx int, cmd string, conn *pubsub.Connection) (*resp.BulkString, bool) {
	val, ok := args.Val[idx].(*resp.BulkString)
	if !ok {
		n := idx + 1
		var pos string
		switch n {
		case 1:
			pos = "1st"
		case 2:
			pos = "2nd"
		case 3:
			pos = "3rd"
		default:
			pos = fmt.Sprintf("%dth", n)
		}
		msg := resp.SimpleError{Val: []byte("wrong data type for the " + pos + " argument of '" + cmd + "' command")}
		conn.Write(&msg)
		return nil, false
	}
	return val, true
}

func getBulkArgMsg(args *resp.Array, idx int, conn *pubsub.Connection, errMsg string) (*resp.BulkString, bool) {
	val, ok := args.Val[idx].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte(errMsg)}
		conn.Write(&msg)
		return nil, false
	}
	return val, true
}

func parseIntBulkArg(args *resp.Array, idx int, cmd string, conn *pubsub.Connection) (int64, bool) {
	bs, ok := getBulkArg(args, idx, cmd, conn)
	if !ok {
		return 0, false
	}
	val, err := strconv.ParseInt(string(bs.Str), 10, 64)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("value is not an integer or out of range")}
		conn.Write(&msg)
		return 0, false
	}
	return val, true
}

func parseFloatBulkArg(args *resp.Array, idx int, cmd string, conn *pubsub.Connection) (float64, bool) {
	bs, ok := getBulkArg(args, idx, cmd, conn)
	if !ok {
		return 0, false
	}
	val, err := strconv.ParseFloat(string(bs.Str), 64)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("value is not a valid float")}
		conn.Write(&msg)
		return 0, false
	}
	return val, true
}
