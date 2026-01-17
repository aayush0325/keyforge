package commands

import (
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func set(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 3 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'set' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 2nd argument of 'set' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	val, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 3rd argument of 'set' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	keyStr := string(key.Str)
	ttl := int64(-1) // default: no expiry
	nx := false

	// Parse optional arguments: NX, EX, PX
	i := 3
	for i < len(args.Val) {
		opt, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{
				Val: []byte("wrong data type for argument"),
			}
			conn.W.Write(msg.ToBytes())
			return
		}

		optStr := strings.ToLower(string(opt.Str))

		switch optStr {
		case "nx":
			nx = true
			i++
		case "ex", "px":
			if i+1 >= len(args.Val) {
				msg := resp.SimpleError{
					Val: []byte("syntax error"),
				}
				conn.W.Write(msg.ToBytes())
				return
			}

			ttlArg, ok := args.Val[i+1].(*resp.BulkString)
			if !ok {
				msg := resp.SimpleError{
					Val: []byte("wrong data type for TTL argument"),
				}
				conn.W.Write(msg.ToBytes())
				return
			}

			parsedTTL, err := strconv.ParseInt(string(ttlArg.Str), 10, 64)
			if err != nil {
				msg := resp.SimpleError{
					Val: []byte("value is not an integer or out of range"),
				}
				conn.W.Write(msg.ToBytes())
				return
			}

			if optStr == "ex" {
				ttl = parsedTTL * 1000 // convert seconds to milliseconds
			} else {
				ttl = parsedTTL
			}
			i += 2
		default:
			msg := resp.SimpleError{
				Val: []byte("syntax error"),
			}
			conn.W.Write(msg.ToBytes())
			return
		}
	}

	channel := make(chan []byte, 1)
	cmd := db.NewCommandWithOptions(keyStr, val.Str, ttl, channel, db.SET, nx)

	// Route to the appropriate shard based on key
	shardCh := db.GetShardChannel(keyStr)
	shardCh <- cmd

	result, ok := <-channel
	if ok {
		conn.W.Write(result)
		close(channel)
	}
}

// setnx implements SETNX command - set if not exists
// Returns 1 if key was set, 0 if key already exists
func setnx(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 3 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'setnx' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 2nd argument of 'setnx' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	val, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type for the 3rd argument of 'setnx' command"),
		}
		conn.W.Write(msg.ToBytes())
		return
	}

	keyStr := string(key.Str)
	channel := make(chan []byte, 1)
	cmd := db.NewCommandWithOptions(keyStr, val.Str, -1, channel, db.SET, true) // nx=true

	shardCh := db.GetShardChannel(keyStr)
	shardCh <- cmd

	result, ok := <-channel
	if ok {
		// Convert SET NX response to SETNX response
		// SET NX returns OK if set, nil if not set
		// SETNX returns 1 if set, 0 if not set
		if len(result) >= 3 && result[0] == '+' { // +OK\r\n
			conn.W.Write([]byte(":1\r\n"))
		} else { // $-1\r\n (nil)
			conn.W.Write([]byte(":0\r\n"))
		}
		close(channel)
	}
}
