package pubsub

import (
	"bufio"
	"log"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// The global pub sub instance is represented by this struct which contains a mapping of each client to
// an array of channels it is subscribed to + channels mapped to an array of connections
type Global struct {
	ChannelToClient map[string]map[*Connection]struct{}
	Mu              sync.RWMutex
	Replicas        []*Connection
	ReplicaMu       sync.Mutex
}

// A "connection" with a client is represented as this struct, this is done to
// keep track of subcribed/unsubscribed modes and number of subscribed channels
type Connection struct {
	W                    *bufio.Writer
	R                    *bufio.Reader
	Channels             map[string]struct{}
	Name                 string        // connection name set by CLIENT SETNAME
	Mu                   sync.Mutex    // protects W for concurrent writes
	IsTransactionQueued  bool          // Denotes whether this connection is in a transaction or not
	IsTransactionRunning bool          // Denotes whether this connection has a running transaction (after EXEC)
	TransactionCommands  []*resp.Array // Commands queued in a transaction after MULTI
	TransactionResponse  *resp.Array   // Response of each transaction commands
	IsMaster             bool          // Denotes whether this connection is to a master
	IsReplica            bool          // Denotes whether this connection is from a replica
	Offset               uint64        // replication offset for this replica
}

// Write writes a message to the connection or queues it if a transaction is executing.
func (c *Connection) Write(msg resp.Message) error {
	if c.IsMaster {
		arr, ok := msg.(*resp.Array)
		if !ok {
			log.Printf("replica didn't write to master 1")
			return nil
		}
		replconf, ok := arr.Val[0].(*resp.BulkString)
		if !ok {
			log.Printf("replica didnt write to master 2")
			return nil
		}
		if strings.ToLower(string(replconf.Str)) == "replconf" {
			_, err := c.W.Write(msg.ToBytes())
			log.Printf("replica wrote to master")
			return err
		}

		log.Printf("replica didnt write to master 3")
		return nil
	}

	if c.IsTransactionRunning {
		if c.TransactionResponse != nil {
			c.TransactionResponse.Val = append(c.TransactionResponse.Val, msg)
		}
		return nil
	}

	_, err := c.W.Write(msg.ToBytes())
	return err
}

var PubSubOnce sync.Once
var Instance Global

func InitPubSub() {
	PubSubOnce.Do(func() {
		Instance = Global{
			ChannelToClient: make(map[string]map[*Connection]struct{}),
		}
	})
}

func (g *Global) AddReplica(conn *Connection) {
	g.ReplicaMu.Lock()
	defer g.ReplicaMu.Unlock()
	g.Replicas = append(g.Replicas, conn)
}

func (g *Global) PropagateToReplicas(cmd []byte) {
	log.Printf("[MASTER] Propagating to replicas: %q", string(cmd))
	g.ReplicaMu.Lock()
	replicas := make([]*Connection, len(g.Replicas))
	copy(replicas, g.Replicas)
	g.ReplicaMu.Unlock()

	for _, replica := range replicas {
		replica.Mu.Lock()
		replica.W.Write(cmd)
		replica.W.Flush()
		replica.Mu.Unlock()
	}
}
