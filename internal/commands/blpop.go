package commands

import (
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func blpop(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 3 {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'blpop' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	// Last argument is timeout
	timeoutString, ok := args.Val[len(args.Val)-1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type for timeout argument of 'blpop' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	timeoutFloat, err := strconv.ParseFloat(string(timeoutString.Str), 64)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("error while parsing timeout argument of 'blpop' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	// Collect all keys (everything except command name and timeout)
	keys := make([]*resp.BulkString, 0, len(args.Val)-2)
	for i := 1; i < len(args.Val)-1; i++ {
		key, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("wrong data type for key argument of 'blpop' command")}
			conn.W.Write(msg.ToBytes())
			return
		}
		keys = append(keys, key)
	}

	// First pass: check if any list has data immediately
	for _, key := range keys {
		list := db.CreateOrGetList(string(key.Str))
		list.Mu.Lock()
		if len(list.Q.Buf) != 0 {
			val, ok := list.Q.PopFront()
			shouldDelete := list.Q.Len() == 0 && list.B.Len() == 0
			list.Mu.Unlock()

			if shouldDelete {
				db.DeleteList(string(key.Str))
				log.Printf("Element and channel queue is empty for list %s, deleting...", key.Str)
			}

			if !ok {
				msg := resp.SimpleError{Val: []byte("unexpectedly list is empty while doing PopFront()")}
				conn.W.Write(msg.ToBytes())
				return
			}

			res := resp.Array{
				Val: []resp.Message{
					&resp.BulkString{Str: key.Str, Size: key.Size},
					&resp.BulkString{Str: []byte(val), Size: len(val)},
				},
			}
			conn.W.Write(res.ToBytes())
			return
		}
		list.Mu.Unlock()
	}

	// No data available, need to block on all keys
	// Register a channel for each list and wait for any to signal
	type listInfo struct {
		key  *resp.BulkString
		list *db.ListEntry
		ch   chan struct{}
	}

	lists := make([]listInfo, len(keys))
	for i, key := range keys {
		list := db.CreateOrGetList(string(key.Str))
		ch := make(chan struct{}, 1)

		list.Mu.Lock()
		list.B.PushFront(ch)
		list.Mu.Unlock()

		lists[i] = listInfo{key: key, list: list, ch: ch}
		log.Printf("Registered blocking channel for list %s", key.Str)
	}

	// cleanup removes channels from all lists except the one that fired (if any)
	cleanup := func(firedIndex int) {
		for i, info := range lists {
			if i == firedIndex {
				continue
			}
			info.list.Mu.Lock()
			info.list.B.Remove(info.ch)
			info.list.Mu.Unlock()
		}
	}

	// waitForAny waits for any channel to fire and returns the index
	waitForAny := func() int {
		// Use reflect.Select for dynamic number of channels
		cases := make([]reflect.SelectCase, len(lists))
		for i, info := range lists {
			cases[i] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(info.ch),
			}
		}
		chosen, _, _ := reflect.Select(cases) // returns the index of the channel that fired first
		return chosen
	}

	if timeoutFloat == 0 {
		// Block indefinitely
		idx := waitForAny()
		cleanup(idx)
		handleChannelEvent(lists[idx].list, conn, lists[idx].key)
		return
	}

	// Block with timeout
	timeoutDuration := time.Duration(timeoutFloat * float64(time.Second))
	timeoutTimer := time.NewTimer(timeoutDuration)
	defer timeoutTimer.Stop()

	// Build select cases: all list channels + timeout
	cases := make([]reflect.SelectCase, len(lists)+1)
	for i, info := range lists {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(info.ch),
		}
	}
	cases[len(lists)] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(timeoutTimer.C),
	}

	chosen, _, _ := reflect.Select(cases)

	if chosen == len(lists) {
		// Timeout fired - need to cleanup and check for race conditions
		for i, info := range lists {
			info.list.Mu.Lock()
			removed := info.list.B.Remove(info.ch)
			info.list.Mu.Unlock()

			if !removed {
				// LPUSH/RPUSH won the race on this list — must consume and use it
				<-info.ch
				// Cleanup remaining lists
				for j := i + 1; j < len(lists); j++ {
					lists[j].list.Mu.Lock()
					lists[j].list.B.Remove(lists[j].ch)
					lists[j].list.Mu.Unlock()
				}
				handleChannelEvent(info.list, conn, info.key)
				return
			}
		}
		// All channels were successfully removed - genuine timeout
		conn.W.Write([]byte("*-1\r\n"))
		return
	}

	// One of the list channels fired
	cleanup(chosen)
	handleChannelEvent(lists[chosen].list, conn, lists[chosen].key)
}

// function to handle the case when the list was empty --> element was added --> blpop removed the event
func handleChannelEvent(list *db.ListEntry, conn *pubsub.Connection, key *resp.BulkString) {
	list.Mu.Lock()
	log.Printf("Lock for list %s acquired by the 'blpop' command goroutine after signal was sent into channel", key.Str)
	val, ok := list.Q.PopFront()
	shouldDelete := list.Q.Len() == 0 && list.B.Len() == 0
	list.Mu.Unlock()

	if shouldDelete {
		db.DeleteList(string(key.Str))
		log.Printf("Element and channel queue is empty for list %s, deleteting...", key.Str)
	}

	log.Printf("Lock for list %s released by the 'blpop' command goroutine", key.Str)

	if !ok {
		msg := resp.SimpleError{Val: []byte("unexpectedly list is empty while doing PopFront()")}
		conn.W.Write(msg.ToBytes())
		return
	}

	res := resp.Array{
		Val: []resp.Message{
			&resp.BulkString{Str: key.Str, Size: key.Size},
			&resp.BulkString{Str: []byte(val), Size: len(val)},
		},
	}

	conn.W.Write(res.ToBytes())
}
