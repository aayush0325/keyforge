package commands

import (
	"bytes"
	"path/filepath"

	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

// ServerConfig holds the configuration parameters for the Redis server
var ServerConfig = map[string]string{
	"dir":        "/tmp",
	"dbfilename": "dump.rdb",
}

func config(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config' command")}
		conn.Write(&msg)
		return
	}

	subCmd, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.Write(&msg)
		return
	}

	switch string(bytes.ToLower(subCmd.Str)) {
	case "get":
		configGet(args, conn)
	case "set":
		configSet(args, conn)
	default:
		msg := resp.SimpleError{Val: []byte("ERR unknown subcommand '" + string(subCmd.Str) + "'. Try CONFIG GET, CONFIG SET.")}
		conn.Write(&msg)
	}
}

func configGet(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 3 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config|get' command")}
		conn.Write(&msg)
		return
	}

	pattern, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.Write(&msg)
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

	conn.Write(result)
}

func configSet(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 4 {
		msg := resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'config|set' command")}
		conn.Write(&msg)
		return
	}

	param, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.Write(&msg)
		return
	}

	value, ok := args.Val[3].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument type")}
		conn.Write(&msg)
		return
	}

	paramStr := string(param.Str)
	valueStr := string(value.Str)

	// Only allow setting known config parameters
	if _, exists := ServerConfig[paramStr]; !exists {
		msg := resp.SimpleError{Val: []byte("ERR unknown configuration parameter '" + paramStr + "'")}
		conn.Write(&msg)
		return
	}

	ServerConfig[paramStr] = valueStr
	conn.Write(&resp.SimpleString{Val: []byte("OK")})
}
