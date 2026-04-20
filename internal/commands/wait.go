package commands

import (
	"log"
	"strconv"
	"time"

	conf "github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func wait(args *resp.Array, conn *pubsub.Connection) {
	// Parse timeout
	timeoutString, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		conn.Write(&resp.SimpleError{Val: []byte("wrong data type for timeout argument of 'wait' command")})
		return
	}

	timeout, err := strconv.ParseInt(string(timeoutString.Str), 10, 64)
	if err != nil {
		conn.Write(&resp.SimpleError{Val: []byte("error while parsing timeout argument of 'wait' command")})
		return
	}

	// Parse numreplicas
	numreplicastr, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		conn.Write(&resp.SimpleError{Val: []byte("wrong data type of 2nd argument for 'wait' command")})
		return
	}

	numreplica, err := strconv.ParseInt(string(numreplicastr.Str), 10, 64)
	if err != nil {
		conn.Write(&resp.SimpleError{Val: []byte("error while parsing numreplicas")})
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
