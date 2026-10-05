package commands

import (
	"bytes"
	"path/filepath"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

// ServerConfig holds the configuration parameters that clients may read and
// write with CONFIG GET / CONFIG SET. "dir" and "dbfilename" are backed by the
// config package because they decide where SAVE and the FULLRESYNC payload come
// from.
var ServerConfig = map[string]string{
	"dir":        conf.Dir,
	"dbfilename": conf.DBFilename,
}

// syncFromConfig refreshes the cached view of the two persistence related
// parameters, which live in the config package.
func syncFromConfig() {
	ServerConfig["dir"] = conf.Dir
	ServerConfig["dbfilename"] = conf.DBFilename
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

	syncFromConfig()

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

	// Only allow setting known config parameters
	if _, exists := ServerConfig[paramStr]; !exists {
		msg := resp.SimpleError{Val: []byte("ERR unknown configuration parameter '" + paramStr + "'")}
		conn.Write(&msg)
		return
	}

	switch paramStr {
	case "dir":
		conf.Dir = string(value.Str)
	case "dbfilename":
		if string(value.Str) == "" {
			conn.Write(&resp.SimpleError{Val: []byte("ERR dbfilename can't be empty")})
			return
		}
		conf.DBFilename = string(value.Str)
	}

	ServerConfig[paramStr] = string(value.Str)
	conn.Write(&resp.SimpleString{Val: []byte("OK")})
}
