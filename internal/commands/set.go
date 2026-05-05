package commands

import (
	"strconv"
	"strings"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func set(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 3, "set", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "set", conn)
	if !ok {
		return
	}

	val, ok := getBulkArg(args, 2, "set", conn)
	if !ok {
		return
	}

	keyStr := string(key.Str)
	ttl := int64(-1) // default: no expiry
	nx := false

	// Parse optional arguments: NX, EX, PX
	i := 3
	for i < len(args.Val) {
		opt, ok := getBulkArgMsg(args, i, conn, "wrong data type for argument")
		if !ok {
			return
		}

		optStr := strings.ToLower(string(opt.Str))

		switch optStr {
		case "nx":
			nx = true
			i++
		case "ex", "px":
			if i+1 >= len(args.Val) {
				if conn != nil {
					msg := resp.SimpleError{
						Val: []byte("syntax error"),
					}
					conn.Write(&msg)
				}
				return
			}

			ttlArg, ok := getBulkArgMsg(args, i+1, conn, "wrong data type for TTL argument")
			if !ok {
				return
			}

			parsedTTL, err := strconv.ParseInt(string(ttlArg.Str), 10, 64)
			if err != nil {
				if conn != nil {
					msg := resp.SimpleError{
						Val: []byte("value is not an integer or out of range"),
					}
					conn.Write(&msg)
				}
				return
			}

			if optStr == "ex" {
				ttl = parsedTTL * 1000 // convert seconds to milliseconds
			} else {
				ttl = parsedTTL
			}
			i += 2
		default:
			if conn != nil {
				msg := resp.SimpleError{
					Val: []byte("syntax error"),
				}
				conn.Write(&msg)
			}
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
		if conn != nil {
			conn.Write(resp.RawMessage(result))
		}
		close(channel)
	}
}

// setnx implements SETNX command - set if not exists
// Returns 1 if key was set, 0 if key already exists
func setnx(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 3, "setnx", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "setnx", conn)
	if !ok {
		return
	}

	val, ok := getBulkArg(args, 2, "setnx", conn)
	if !ok {
		return
	}

	keyStr := string(key.Str)
	channel := make(chan []byte, 1)
	cmd := db.NewCommandWithOptions(keyStr, val.Str, -1, channel, db.SET, true) // nx=true

	shardCh := db.GetShardChannel(keyStr)
	shardCh <- cmd

	result, ok := <-channel
	if ok {
		if conn != nil {
			// Convert SET NX response to SETNX response
			// SET NX returns OK if set, nil if not set
			// SETNX returns 1 if set, 0 if not set
			if len(result) >= 3 && result[0] == '+' { // +OK\r\n
				conn.Write(&resp.Integer{Val: 1})
			} else { // $-1\r\n (nil)
				conn.Write(&resp.Integer{Val: 0})
			}
		}
		close(channel)
	}
}
