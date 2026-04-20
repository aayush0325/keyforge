package commands

import (
	"fmt"
	"strconv"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/rdb"
	"github.com/aayush0325/keyforge/internal/resp"
)

func psync(_ *resp.Array, conn *pubsub.Connection) {
	response := fmt.Sprintf("FULLRESYNC %s 0", conf.ReplID)
	conn.Write(&resp.SimpleString{Val: []byte(response)})

	rdbLength := strconv.Itoa(len(rdb.EmptyRDB))
	rdbData := "$" + rdbLength + "\r\n" + string(rdb.EmptyRDB)
	conn.W.Write([]byte(rdbData))
	conn.W.Flush()

	conn.IsReplica = true
	pubsub.Instance.AddReplica(conn)
}
