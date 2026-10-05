package commands

import (
	"fmt"
	"log"
	"strconv"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/rdb"
	"github.com/aayush0325/keyforge/internal/resp"
)

// psync handles the PSYNC command a replica sends after the handshake.
//
// The master answers with +FULLRESYNC <replid> <offset> followed by the whole
// dataset as an RDB payload, exactly like Redis does. The snapshot is taken
// while write commands are held back, so a write can never end up both inside
// the payload and on the replication stream.
func psync(_ *resp.Array, conn *pubsub.Connection) {
	var payload []byte

	err := withDatasetLock(conn, func() error {
		var err error
		payload, err = rdb.Serialize()
		return err
	})
	if err != nil {
		log.Printf("psync: failed serializing the dataset: %v", err)
		conn.Write(&resp.SimpleError{Val: []byte("ERR can't serialize the dataset")})
		return
	}

	// The offset that is reported is the one the replica continues from: it
	// only accounts for the replication stream, the snapshot itself is not part
	// of it. keyforge does not keep a replication backlog, so both sides simply
	// count the bytes of the commands that follow, which keeps their offsets in
	// sync.
	offset := conf.GetOffset()

	response := fmt.Sprintf("FULLRESYNC %s %d", conf.ReplID, offset)
	log.Printf("psync: starting full resynchronization with replica, offset %d, payload %d bytes", offset, len(payload))

	conn.Write(&resp.SimpleString{Val: []byte(response)})

	// The payload is sent as a bulk string, exactly like Redis does.
	conn.Mu.Lock()
	conn.W.Write([]byte("$" + strconv.Itoa(len(payload)) + "\r\n"))
	conn.W.Write(payload)
	conn.W.Write([]byte("\r\n"))
	conn.W.Flush()
	conn.Mu.Unlock()

	// From now on every write command is streamed to this connection.
	conn.IsReplica = true
	conn.Offset = offset

	pubsub.Instance.AddReplica(conn)
}
