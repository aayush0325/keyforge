package commands

import (
	"log"
	"time"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func wait(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 3, "wait", conn) {
		return
	}

	numreplica, ok := parseIntBulkArg(args, 1, "wait", conn)
	if !ok {
		return
	}

	timeout, ok := parseIntBulkArg(args, 2, "wait", conn)
	if !ok {
		return
	}

	log.Printf("offset: %d, lastconfirmed: %d, timeout: %d", conf.Offset, conf.LastConfirmedOffset, timeout)

	// Fast path
	if conf.Offset <= conf.LastConfirmedOffset {
		pubsub.Instance.ReplicaMu.Lock()
		count := len(pubsub.Instance.Replicas)
		pubsub.Instance.ReplicaMu.Unlock()

		conn.Write(&resp.Integer{Val: int64(count)})
		return
	}

	// Copy replicas safely
	pubsub.Instance.ReplicaMu.Lock()
	replicas := make([]*pubsub.Connection, len(pubsub.Instance.Replicas))
	copy(replicas, pubsub.Instance.Replicas)
	pubsub.Instance.ReplicaMu.Unlock()

	getack := &resp.Array{
		Val: []resp.Message{
			&resp.BulkString{Str: []byte("REPLCONF"), Size: 8},
			&resp.BulkString{Str: []byte("GETACK"), Size: 6},
			&resp.BulkString{Str: []byte("*"), Size: 1},
		},
	}

	pubsub.Instance.PropagateToReplicas(getack.ToBytes())

	deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)

	for {
		cnt := int64(0)

		for _, replica := range replicas {
			if uint64(replica.Offset) >= conf.Offset {
				cnt++
			}
		}

		if cnt >= numreplica {
			conf.LastConfirmedOffset = conf.Offset
			conn.Write(&resp.Integer{Val: cnt})
			conf.Offset += uint64(len(getack.ToBytes()))

			return
		}

		if time.Now().After(deadline) && timeout != 0 {
			conf.LastConfirmedOffset = conf.Offset
			conn.Write(&resp.Integer{Val: cnt})
			conf.Offset += uint64(len(getack.ToBytes()))

			return
		}

	}

}
