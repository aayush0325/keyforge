package commands

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func replconf(arr *resp.Array, conn *pubsub.Connection) {
	if !conf.IsReplica {
		// The instance is a replica and replying to a master
		sub, ok := arr.Val[1].(*resp.BulkString)
		if !ok {
			conn.Write(&resp.SimpleError{Val: []byte("type mismatch in 2nd argument of replconf")})
			return
		}

		if strings.ToLower(string(sub.Str)) == "ack" {
			offsetstr, ok := arr.Val[2].(*resp.BulkString)
			if !ok {
				conn.Write(&resp.SimpleError{Val: []byte("type mismatch in 3nd argument of replconf ack")})
				return
			}

			offset, err := strconv.ParseUint(string(offsetstr.Str), 10, 64)
			if err != nil {
				conn.Write(&resp.SimpleError{Val: []byte("type mismatch in 3nd argument of replconf ack")})
				return
			}

			conn.Offset = offset
		} else {
			conn.Write(resp.RawMessage("+OK\r\n"))
			log.Printf("Masters offset is: %d", conf.Offset)
			return
		}
	} else {
		// The instance is a replica and replying to a master
		sub, ok := arr.Val[1].(*resp.BulkString)
		if !ok {
			conn.Write(&resp.SimpleError{Val: []byte("type mismatch in 2nd argument of replconf")})
			return
		}

		// master is asking replica the offset
		if strings.ToLower(string(sub.Str)) == "getack" {
			ackResp := &resp.Array{
				Val: []resp.Message{
					&resp.BulkString{Str: []byte("REPLCONF"), Size: 8},
					&resp.BulkString{Str: []byte("ACK"), Size: 3},
					&resp.BulkString{Str: []byte(fmt.Sprintf("%d", conf.Offset)), Size: len(fmt.Sprintf("%d", conf.Offset))},
				},
			}

			log.Printf("replconf called by master to replica")

			conn.Write(ackResp)
		}
	}
}
