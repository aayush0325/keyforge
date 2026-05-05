package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
	"github.com/aayush0325/keyforge/internal/streams"
)

func xrange(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 4, "xrange", conn) {
		return
	}

	streamKey, ok := getBulkArg(args, 1, "xrange", conn)
	if !ok {
		return
	}

	startIDStr, ok := getBulkArg(args, 2, "xrange", conn)
	if !ok {
		return
	}

	endIDStr, ok := getBulkArg(args, 3, "xrange", conn)
	if !ok {
		return
	}

	// Parse start ID (sequence defaults to 0)
	startID, err := streams.NewStreamIDForRange(string(startIDStr.Str), false)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("ERR Invalid stream ID for 'xrange' command")}
		conn.Write(&msg)
		return
	}

	// Parse end ID (sequence defaults to max uint64)
	endID, err := streams.NewStreamIDForRange(string(endIDStr.Str), true)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("ERR Invalid stream ID for 'xrange' command")}
		conn.Write(&msg)
		return
	}

	streams.Global.Mu.Lock()
	stream, exists := streams.Global.KV[string(streamKey.Str)]
	if !exists {
		streams.Global.Mu.Unlock()
		// Return empty array if stream doesn't exist
		emptyArr := &resp.Array{Val: []resp.Message{}}
		conn.Write(emptyArr)
		return
	}

	// Get entries in range
	entries := stream.Range(startID, endID)
	streams.Global.Mu.Unlock()

	// Build response: array of [id, [key1, val1, key2, val2, ...]]
	resultArr := make([]resp.Message, 0, len(entries))
	for _, entry := range entries {
		// Build the key-value array
		kvArr := make([]resp.Message, 0, len(entry.Entry)*2)
		for k, v := range entry.Entry {
			kvArr = append(kvArr, &resp.BulkString{Str: []byte(k), Size: len(k)})
			kvArr = append(kvArr, &resp.BulkString{Str: []byte(v), Size: len(v)})
		}

		// Build the entry array: [id, [key-value pairs]]
		idStr := entry.ID.String()
		entryArr := &resp.Array{
			Val: []resp.Message{
				&resp.BulkString{Str: []byte(idStr), Size: len(idStr)},
				&resp.Array{Val: kvArr},
			},
		}
		resultArr = append(resultArr, entryArr)
	}

	result := &resp.Array{Val: resultArr}
	conn.Write(result)
}
