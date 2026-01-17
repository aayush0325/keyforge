package commands

import (
	"bytes"
	"path/filepath"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// ServerConfig holds the configuration parameters for the Redis server
var ServerConfig = map[string]string{
	"dir":        "/tmp",
	"dbfilename": "dump.rdb",
}

func config(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	subCmd, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.W.Write(msg.ToBytes())
		return
	}

	switch string(bytes.ToLower(subCmd.Str)) {
	case "get":
		configGet(args, conn)
	case "set":
		configSet(args, conn)
	default:
		msg := resp.SimpleError{Val: []byte("ERR unknown subcommand '" + string(subCmd.Str) + "'. Try CONFIG GET, CONFIG SET.")}
		conn.W.Write(msg.ToBytes())
	}
}

func configGet(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 3 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config|get' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	pattern, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.W.Write(msg.ToBytes())
		return
	}

	patternStr := string(pattern.Str)
	result := &resp.Array{Val: []resp.Message{}}

	for key, value := range ServerConfig {
		matched, err := filepath.Match(patternStr, key)
		if err != nil {
			continue
		}
		if matched {
			keyBulk := &resp.BulkString{Str: []byte(key), Size: len(key)}
			valueBulk := &resp.BulkString{Str: []byte(value), Size: len(value)}
			result.Val = append(result.Val, keyBulk, valueBulk)
		}
	}

	conn.W.Write(result.ToBytes())
}

func configSet(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 4 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config|set' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	param, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.W.Write(msg.ToBytes())
		return
	}

	value, ok := args.Val[3].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.W.Write(msg.ToBytes())
		return
	}

	paramStr := string(param.Str)
	valueStr := string(value.Str)

	// Only allow setting known config parameters
	if _, exists := ServerConfig[paramStr]; !exists {
		msg := resp.SimpleError{Val: []byte("ERR unknown configuration parameter '" + paramStr + "'")}
		conn.W.Write(msg.ToBytes())
		return
	}

	ServerConfig[paramStr] = valueStr
	conn.W.Write([]byte("+OK\r\n"))
}
