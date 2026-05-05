package commands

import (
	"log"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func lpush(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 2, "lpush", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "lpush", conn)
	if !ok {
		return
	}

	list := db.CreateOrGetList(string(key.Str))

	list.Mu.Lock()
	log.Printf("Lock for list %s acquired by the 'lpush' command goroutine", key.Str)

	for i := 2; i < len(args.Val); i++ {
		val, ok := getBulkArgMsg(args, i, conn, "wrong data type of list entry in 'lpush' command")
		if !ok {
			return
		}
		list.Q.PushFront(string(val.Str))
	}

	res := resp.Integer{Val: int64(list.Q.Len())}
	ch, ok := list.B.PopBack()
	list.Mu.Unlock()
	log.Printf("Lock for list %s released by the 'lpush' command goroutine", key.Str)

	if ok {
		ch <- struct{}{}
	}

	conn.Write(&res)
}
