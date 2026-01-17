package commands

import (
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/streams"
)

func xadd(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 5 {
		msg := resp.SimpleError{Val: []byte("too few arguments for 'xadd' command")}
		conn.W.Write(msg.ToBytes())
		return
	}
	streamKey, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument for 'xadd' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	rawStreamID, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("ERR invalid argument for 'xadd' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	streamID, err := streams.NewStreamID(string(rawStreamID.Str))
	if err != nil {
		msg := resp.SimpleError{Val: []byte("Invalid stream ID for 'xadd' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	hashmap := make(map[string]string)
	for i := 3; i < len(args.Val); i += 2 {
		if i+1 > len(args.Val) {
			msg := resp.SimpleError{Val: []byte("Invalid number of args for 'xadd' command")}
			conn.W.Write(msg.ToBytes())
			return
		}
		key, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("ERR invalid argument for 'xadd' command")}
			conn.W.Write(msg.ToBytes())
			return
		}

		val, ok := args.Val[i+1].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("ERR invalid argument for 'xadd' command")}
			conn.W.Write(msg.ToBytes())
			return
		}

		hashmap[string(key.Str)] = string(val.Str)
	}

	streams.Global.Mu.Lock()
	existingStream, streamExists := streams.Global.KV[string(streamKey.Str)]

	// Handle auto-generation of time part (when ID is just "*")
	if streamID.AutoMs {
		streamID.Ms = uint64(time.Now().UnixMilli())
	}

	// Handle auto-sequence generation
	if streamID.AutoSeq {
		if !streamExists || existingStream.LastEntry == nil {
			// Stream is empty or doesn't exist
			if streamID.Ms == 0 {
				// Special case: if time part is 0, sequence starts at 1
				streamID.Seq = 1
			} else {
				// Otherwise sequence starts at 0
				streamID.Seq = 0
			}
		} else {
			// Stream has entries
			lastEntry := existingStream.LastEntry
			if lastEntry.ID.Ms == streamID.Ms {
				// Same time part, increment sequence
				streamID.Seq = lastEntry.ID.Seq + 1
			} else {
				// Different time part
				if streamID.Ms == 0 {
					streamID.Seq = 1
				} else {
					streamID.Seq = 0
				}
			}
		}
	}

	streamEntry := &streams.StreamEntry{
		ID:    streamID,
		Entry: hashmap,
	}

	// Validate entry ID: 0-0 is always invalid
	if streamID.IsZero() {
		streams.Global.Mu.Unlock()
		msg := resp.SimpleError{Val: []byte("ERR The ID specified in XADD must be greater than 0-0")}
		conn.W.Write(msg.ToBytes())
		return
	}

	// Generate the actual ID string to return
	actualIDStr := streamID.String()
	actualIDBulk := &resp.BulkString{Str: []byte(actualIDStr), Size: len(actualIDStr)}

	if !streamExists {
		// New stream - just create it (ID already validated to be > 0-0)
		streams.Global.KV[string(streamKey.Str)] = streams.NewStream(streamEntry)
		streams.Global.Mu.Unlock()
		conn.W.Write(actualIDBulk.ToBytes())
		return
	}

	// Existing stream - validate that new ID > last entry's ID
	if existingStream.LastEntry != nil && streamID.Compare(existingStream.LastEntry.ID) <= 0 {
		streams.Global.Mu.Unlock()
		msg := resp.SimpleError{Val: []byte("ERR The ID specified in XADD is equal or smaller than the target stream top item")}
		conn.W.Write(msg.ToBytes())
		return
	}

	existingStream.Insert(streamEntry, []byte(actualIDStr))

	// Notify blocking listeners
	for _, listener := range existingStream.BlockingListeners {
		if streamID.Compare(listener.WaitingID) > 0 {
			select {
			case listener.Channel <- struct{}{}:
			default:
			}
		}
	}

	streams.Global.Mu.Unlock()
	conn.W.Write(actualIDBulk.ToBytes())
}
