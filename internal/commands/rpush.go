package commands

import (
	"log"

	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func rpush(args *resp.Array, conn *pubsub.Connection) {
	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 1st argument for 'rpush' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	list := db.CreateOrGetList(string(key.Str))

	list.Mu.Lock()
	log.Printf("Lock for list %s acquired by the 'rpush' command goroutine", key.Str)

	for i := 2; i < len(args.Val); i++ {
		val, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("wrong data type of list entry in 'rpush' command")}
			conn.W.Write(msg.ToBytes())
			list.Mu.Unlock()
			log.Printf("Lock for list %s released by the 'rpush' command goroutine", key.Str)
			return
		}
		list.Q.PushBack(string(val.Str))
	}

	res := resp.Integer{Val: int64(list.Q.Len())}
	ch, ok := list.B.PopBack()
	list.Mu.Unlock()
	log.Printf("Lock for list %s released by the 'rpush' command goroutine", key.Str)

	if ok {
		ch <- struct{}{}
	}

	conn.W.Write(res.ToBytes())
}
