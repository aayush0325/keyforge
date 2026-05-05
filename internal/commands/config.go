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
	if !requireMinArgs(args, 2, "config", conn) {
		return
	}

	subCmd, ok := getBulkArg(args, 1, "config", conn)
	if !ok {
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
	if !requireMinArgs(args, 3, "config|get", conn) {
		return
	}

	pattern, ok := getBulkArg(args, 2, "config|get", conn)
	if !ok {
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
	if !requireMinArgs(args, 4, "config|set", conn) {
		return
	}

	param, ok := getBulkArg(args, 2, "config|set", conn)
	if !ok {
		return
	}

	value, ok := getBulkArg(args, 3, "config|set", conn)
	if !ok {
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
