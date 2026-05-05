package commands

import (
	"log"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func lpop(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 || len(args.Val) > 3 {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'lpop' command")}
		conn.Write(&msg)
		return
	}

	key, ok := getBulkArg(args, 1, "lpop", conn)
	if !ok {
		return
	}
	num := int64(1)

	if len(args.Val) == 3 {
		var ok bool
		num, ok = parseIntBulkArg(args, 2, "lpop", conn)
		if !ok {
			return
		}
	}

	list := db.GetList(string(key.Str))
	if list == nil {
		conn.Write(&resp.BulkString{Str: nil, Size: -1})
		return
	}
	res := resp.Array{Val: make([]resp.Message, 0)}

	list.Mu.Lock()
	log.Printf("Lock for list %s acquired by the 'lpop' command goroutine", key.Str)

	for i := int64(0); i < num; i++ {
		val, ok := list.Q.PopFront()
		if ok {
			element := &resp.BulkString{Str: []byte(val), Size: len(val)}
			res.Val = append(res.Val, element)
		}
	}

	shouldDelete := list.Q.Len() == 0 && list.B.Len() == 0
	list.Mu.Unlock()

	log.Printf("Lock for list %s released by the 'lpop' command goroutine", key.Str)
	if shouldDelete {
		db.DeleteList(string(key.Str))
		log.Printf("Element and channel queue is empty for list %s, deleting...", key.Str)
	}

	if len(args.Val) == 2 {
		conn.Write(res.Val[0])
		return
	}
	conn.Write(&res)
}
