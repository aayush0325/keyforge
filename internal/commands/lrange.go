package commands

import (
	"log"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
	"github.com/aayush0325/keyforge/internal/utils"
)

func lrange(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 4, "lrange", conn) {
		return
	}

	key, ok := getBulkArg(args, 1, "lrange", conn)
	if !ok {
		return
	}
	start, ok := parseIntBulkArg(args, 2, "lrange", conn)
	if !ok {
		return
	}

	stop, ok := parseIntBulkArg(args, 3, "lrange", conn)
	if !ok {
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
