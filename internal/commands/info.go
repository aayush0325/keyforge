package commands

import (
	"fmt"
	"strings"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func info(args *resp.Array, conn *pubsub.Connection) {
	section := ""
	if len(args.Val) > 1 {
		str, ok := args.Val[1].(*resp.BulkString)
		if ok {
			section = strings.ToLower(string(str.Str))
		}
	}

	var output string

	switch section {
	case "replication":
		if conf.IsReplica {
			output = "role:slave"
		} else {
			output = fmt.Sprintf("role:master\r\nmaster_replid:%s\r\nmaster_repl_offset:%d", conf.ReplID, conf.Offset)
		}
	default:
		if conf.IsReplica {
			output = "role:slave"
		} else {
			output = fmt.Sprintf("role:master\r\nmaster_replid:%s\r\nmaster_repl_offset:%d", conf.ReplID, conf.Offset)
		}
	}

	conn.Write(&resp.BulkString{Str: []byte(output), Size: len(output)})
}
