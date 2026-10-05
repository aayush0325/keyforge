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

// replconf handles the REPLCONF command in both directions.
//
// On a master it receives REPLCONF ACK <offset> from the replicas, which is how
// WAIT learns how far they got. On a replica it answers the REPLCONF GETACK *
// sent by the master with REPLCONF ACK <offset>.
func replconf(arr *resp.Array, conn *pubsub.Connection) {
	if len(arr.Val) < 2 {
		conn.Write(&resp.SimpleError{Val: []byte("ERR wrong number of arguments for 'replconf' command")})
		return
	}

	sub, ok := arr.Val[1].(*resp.BulkString)
	if !ok {
		conn.Write(&resp.SimpleError{Val: []byte("ERR type mismatch in 2nd argument of replconf")})
		return
	}

	option := strings.ToLower(string(sub.Str))

	switch option {
	case "ack":
		offsetstr, ok := argAsBulkString(arr, 2)
		if !ok {
			conn.Write(&resp.SimpleError{Val: []byte("ERR type mismatch in 3nd argument of replconf ack")})
			return
		}

		offset, err := strconv.ParseUint(string(offsetstr), 10, 64)
		if err != nil {
			conn.Write(&resp.SimpleError{Val: []byte("ERR value is not an integer or out of range")})
			return
		}

		if conf.IsReplica {
			// The master tells us how far it got, remember it for WAIT
			conf.SetLastConfirmedOffset(offset)
			return
		}

		// A replica confirmed it applied everything up to this offset
		conn.Offset = offset
		log.Printf("replica %s acknowledged offset %d", conn.Name, offset)

	case "getack":
		// The master wants to know our offset, answer with REPLCONF ACK
		offset := fmt.Sprintf("%d", conf.GetOffset())
		conn.Write(&resp.Array{
			Val: []resp.Message{
				&resp.BulkString{Str: []byte("REPLCONF"), Size: 8},
				&resp.BulkString{Str: []byte("ACK"), Size: 3},
				&resp.BulkString{Str: []byte(offset), Size: len(offset)},
			},
		})

	case "listening-port", "ip-address", "capa", "version":
		// Handshake chatter, acknowledged so that the master moves on to PSYNC
		conn.Write(resp.RawMessage("+OK\r\n"))

	default:
		log.Printf("replconf called with unknown option %q", option)
		conn.Write(resp.RawMessage("+OK\r\n"))
	}
}

func argAsBulkString(arr *resp.Array, idx int) ([]byte, bool) {
	if idx >= len(arr.Val) {
		return nil, false
	}
	value, ok := arr.Val[idx].(*resp.BulkString)
	if !ok {
		return nil, false
	}
	return value.Str, true
}
