package commands

import (
	"log"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func lpop(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 3 && len(args.Val) != 2 {
		msg := resp.SimpleError{Val: []byte("wrong number of arguments for 'lpop' command")}
		conn.W.Write(msg.ToBytes())
		return
	}

	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 1st argument for 'lpop' command")}
		conn.W.Write(msg.ToBytes())
		return
	}
	num := int64(1)

	if len(args.Val) == 3 {
		numberString, ok := args.Val[2].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{Val: []byte("wrong data type of 3rd argument for 'lpop' command")}
			conn.W.Write(msg.ToBytes())
			return
		}
		number, err := strconv.ParseInt(string(numberString.Str), 10, 64)
		if err != nil {
			msg := resp.SimpleError{Val: []byte("error while parsing the 3rd argument of 'lpop' command")}
			conn.W.Write(msg.ToBytes())
			return
		}
		num = number
	}

	list := db.GetList(string(key.Str))
	if list == nil {
		conn.W.Write([]byte(resp.NULLBULKSTRING))
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
		conn.W.Write(res.Val[0].ToBytes())
		return
	}
	conn.W.Write(res.ToBytes())
}
