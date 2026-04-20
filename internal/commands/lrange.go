package commands

import (
	"log"
	"strconv"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
	"github.com/aayush0325/keyforge/internal/utils"
)

func lrange(args *resp.Array, conn *pubsub.Connection) {
	key, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 1st argument for 'lrange' command")}
		conn.Write(&msg)
		return
	}
	startString, ok := args.Val[2].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 2nd argument for 'lrange' command")}
		conn.Write(&msg)
		return
	}

	stopString, ok := args.Val[3].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{Val: []byte("wrong data type of 3rd argument for 'lrange' command")}
		conn.Write(&msg)
		return
	}

	start, err := strconv.ParseInt(string(startString.Str), 10, 64)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("error while parsing the start index")}
		conn.Write(&msg)
		return
	}
	stop, err := strconv.ParseInt(string(stopString.Str), 10, 64)
	if err != nil {
		msg := resp.SimpleError{Val: []byte("error while parsing the stop index")}
		conn.Write(&msg)
		return
	}

	list := db.GetList(string(key.Str))
	if list == nil {
		res := resp.Array{Val: make([]resp.Message, 0)}
		conn.Write(&res)
		return
	}
	list.Mu.Lock()

	// Check if indices are valid
	if !utils.ValidateIndices(start, stop, uint(len(list.Q.Buf))) {
		res := resp.Array{Val: make([]resp.Message, 0)}
		conn.Write(&res)
		list.Mu.Unlock()
		log.Printf("Lock for list %s released by the 'lrange' command goroutine", key.Str)
		return
	}

	// No need to validate indices here as that is already done above, we can assume that these are valid indices
	start, _ = utils.GetPositiveIndex(uint(len(list.Q.Buf)), start)
	stop, _ = utils.GetPositiveIndex(uint(len(list.Q.Buf)), stop)

	slice := list.Q.Buf[start : stop+1] // stop index is included in this slice
	res := utils.GetRespArrayBulkString(slice)
	list.Mu.Unlock()
	log.Printf("Lock for list %s released by the 'lrange' command goroutine", key.Str)
	conn.Write(&res)
}
