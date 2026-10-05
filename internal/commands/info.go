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
		output = replicationInfo()
	default:
		if conf.IsReplica {
			output = "role:slave"
		} else {
			output = replicationInfo()
		}
	}

	conn.Write(&resp.BulkString{Str: []byte(output), Size: len(output)})
}

// replicationInfo renders the INFO replication section.
func replicationInfo() string {
	if conf.IsReplica {
		return fmt.Sprintf("role:slave\r\nmaster_replid:%s\r\nmaster_repl_offset:%d", conf.ReplID, conf.GetOffset())
	}

	pubsub.Instance.ReplicaMu.Lock()
	replicas := len(pubsub.Instance.Replicas)
	pubsub.Instance.ReplicaMu.Unlock()

	return fmt.Sprintf(
		"role:master\r\nmaster_replid:%s\r\nmaster_repl_offset:%d\r\nconnected_slaves:%d",
		conf.ReplID, conf.GetOffset(), replicas)
}
