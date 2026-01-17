package commands

import (
	"bytes"
	"reflect"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/streams"
)

func xread(args *resp.Array, conn *pubsub.Connection) {
	// XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] id [id ...]

	blockMs := int64(-1)
	streamsIdx := -1

	for i := 0; i < len(args.Val); i++ {
		arg := args.Val[i]
		if bs, ok := arg.(*resp.BulkString); ok {
			argStr := string(bytes.ToLower(bs.Str))
			if argStr == "streams" {
				streamsIdx = i
				break
			}
			if argStr == "block" {
				if i+1 >= len(args.Val) {
					msg := resp.SimpleError{Val: []byte("ERR syntax error")}
					conn.W.Write(msg.ToBytes())
					return
				}
				blockArg, ok := args.Val[i+1].(*resp.BulkString)
				if !ok {
					msg := resp.SimpleError{Val: []byte("ERR syntax error")}
					conn.W.Write(msg.ToBytes())
					return
				}
				parsedBlockMs, err := strconv.ParseInt(string(blockArg.Str), 10, 64)
				if err != nil {
					msg := resp.SimpleError{Val: []byte("ERR value is not an integer or out of range")}
					conn.W.Write(msg.ToBytes())
					return
				}
				blockMs = parsedBlockMs
				i++ // Skip the value
			}
		}
	}

	if streamsIdx == -1 || streamsIdx+1 >= len(args.Val) {
		msg := resp.SimpleError{Val: []byte("ERR syntax error")}
		conn.W.Write(msg.ToBytes())
		return
	}

	numStreams := (len(args.Val) - (streamsIdx + 1)) / 2
	if (len(args.Val)-(streamsIdx+1))%2 != 0 {
		msg := resp.SimpleError{Val: []byte("ERR syntax error")}
		conn.W.Write(msg.ToBytes())
		return
	}

	keys := make([]string, numStreams)
	ids := make([]*streams.StreamID, numStreams)

	for i := 0; i < numStreams; i++ {
		keyBS, ok := args.Val[streamsIdx+1+i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("ERR syntax error")}
			conn.W.Write(msg.ToBytes())
			return
		}
		keys[i] = string(keyBS.Str)
	}

	rawIDs := make([]string, numStreams)
	for i := 0; i < numStreams; i++ {
		idBS, ok := args.Val[streamsIdx+1+numStreams+i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("ERR syntax error")}
			conn.W.Write(msg.ToBytes())
			return
		}
		rawIDs[i] = string(idBS.Str)
	}

	streams.Global.Mu.Lock()

	// Resolve IDs (especially $)
	for i := 0; i < numStreams; i++ {
		if rawIDs[i] == "$" {
			stream, exists := streams.Global.KV[keys[i]]
			if exists && stream.LastEntry != nil {
				ids[i] = stream.LastEntry.ID
			} else {
				ids[i] = &streams.StreamID{Ms: 0, Seq: 0}
			}
		} else {
			id, err := streams.NewStreamID(rawIDs[i])
			if err != nil {
				streams.Global.Mu.Unlock()
				msg := resp.SimpleError{Val: []byte("ERR Invalid stream ID")}
				conn.W.Write(msg.ToBytes())
				return
			}
			ids[i] = id
		}
	}
	// Helper to fetch data
	fetchData := func() []resp.Message {
		var responseStreams []resp.Message
		for i := 0; i < numStreams; i++ {
			key := keys[i]
			startID := ids[i]

			stream, exists := streams.Global.KV[key]
			if !exists {
				continue
			}

			maxID := &streams.StreamID{Ms: ^uint64(0), Seq: ^uint64(0)}
			allEntries := stream.Range(startID, maxID)

			var filteredEntries []resp.Message
			for _, entry := range allEntries {
				if entry.ID.Compare(startID) > 0 {
					kvPairs := make([]resp.Message, 0, len(entry.Entry)*2)
					for k, v := range entry.Entry {
						kvPairs = append(kvPairs, &resp.BulkString{Str: []byte(k), Size: len(k)})
						kvPairs = append(kvPairs, &resp.BulkString{Str: []byte(v), Size: len(v)})
					}

					idStr := entry.ID.String()
					entryArr := &resp.Array{
						Val: []resp.Message{
							&resp.BulkString{Str: []byte(idStr), Size: len(idStr)},
							&resp.Array{Val: kvPairs},
						},
					}
					filteredEntries = append(filteredEntries, entryArr)
				}
			}

			if len(filteredEntries) > 0 {
				streamRes := &resp.Array{
					Val: []resp.Message{
						&resp.BulkString{Str: []byte(key), Size: len(key)},
						&resp.Array{Val: filteredEntries},
					},
				}
				responseStreams = append(responseStreams, streamRes)
			}
		}
		return responseStreams
	}

	data := fetchData()
	if len(data) > 0 || blockMs == -1 {
		streams.Global.Mu.Unlock()
		if len(data) == 0 {
			conn.W.Write([]byte("*-1\r\n"))
		} else {
			finalResponse := &resp.Array{Val: data}
			conn.W.Write(finalResponse.ToBytes())
		}
		return
	}

	// Blocking logic
	// Register listeners
	channels := make([]chan struct{}, 0, numStreams)
	listeners := make([]*streams.BlockingListener, 0, numStreams)

	for i := 0; i < numStreams; i++ {
		key := keys[i]
		startID := ids[i]

		// Ensure stream exists or create it?
		// Redis XREAD BLOCK creates empty streams if they don't exist?
		// Actually, XREAD BLOCK waits on keys. If key doesn't exist, it waits for it to be created.
		// So we need to put a listener on the key in Global.KV.
		// But our Global.KV is a map. If the key is missing, we can't attach a listener to the Stream struct.
		// We might need to create an empty Stream struct if it doesn't exist.

		stream, exists := streams.Global.KV[key]
		if !exists {
			stream = streams.NewEmptyStream()
			streams.Global.KV[key] = stream
		}

		ch := make(chan struct{}, 1)
		listener := &streams.BlockingListener{
			Channel:   ch,
			WaitingID: startID,
		}
		stream.BlockingListeners = append(stream.BlockingListeners, listener)
		listeners = append(listeners, listener)
		channels = append(channels, ch)
	}
	streams.Global.Mu.Unlock()

	// Wait
	// Build select cases
	cases := make([]reflect.SelectCase, len(channels)+1)
	for i, ch := range channels {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
	}

	// Timeout case
	var timeoutCh <-chan time.Time
	if blockMs > 0 {
		timer := time.NewTimer(time.Duration(blockMs) * time.Millisecond)
		defer timer.Stop()
		timeoutCh = timer.C
	} else {
		// blockMs = 0 means block indefinitely
		// We can use a nil channel which blocks forever in select
	}

	cases[len(channels)] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(timeoutCh),
	}
	if blockMs == 0 {
		// If 0, we don't want a timeout case that fires immediately (nil chan blocks, but let's be explicit)
		// reflect.Select with nil chan blocks forever.
		// But wait, if timeoutCh is nil, SelectRecv on it will block forever. Correct.
	}

	chosen, _, _ := reflect.Select(cases)

	// Cleanup listeners
	streams.Global.Mu.Lock()
	for i := 0; i < numStreams; i++ {
		key := keys[i]
		if stream, exists := streams.Global.KV[key]; exists {
			// Remove listener
			// We need to find our listener and remove it.
			// Since we have the pointer, we can filter.
			newListeners := make([]*streams.BlockingListener, 0, len(stream.BlockingListeners))
			for _, l := range stream.BlockingListeners {
				if l != listeners[i] {
					newListeners = append(newListeners, l)
				}
			}
			stream.BlockingListeners = newListeners
		}
	}

	if chosen == len(channels) {
		// Timeout
		streams.Global.Mu.Unlock()
		conn.W.Write([]byte("*-1\r\n"))
		return
	}

	// Data arrived
	data = fetchData()
	streams.Global.Mu.Unlock()

	if len(data) == 0 {
		// Should not happen if signaled correctly, but possible race?
		// If race, maybe we should return null or try again?
		// Redis behavior: if signaled, it returns data.
		// If we woke up but data is gone (unlikely with append-only streams), return null.
		conn.W.Write([]byte("*-1\r\n"))
	} else {
		finalResponse := &resp.Array{Val: data}
		conn.W.Write(finalResponse.ToBytes())
	}
}
