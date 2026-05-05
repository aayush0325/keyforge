package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func unsubscribe(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 2, "unsubscribe", conn) {
		return
	}

	channel, ok := getBulkArg(args, 1, "unsubscribe", conn)
	if !ok {
		return
	}
	delete(conn.Channels, string(channel.Str)) // unlink the channel from the connection struct
	pubsub.Instance.Mu.Lock()
	// unlink the connection from the channel to client mapping IF it exists
	if _, ok := pubsub.Instance.ChannelToClient[string(channel.Str)]; ok {
		delete(pubsub.Instance.ChannelToClient[string(channel.Str)], conn)
	}
	pubsub.Instance.Mu.Unlock()

	res := resp.Array{
		Val: []resp.Message{
			&resp.BulkString{Str: []byte("unsubscribe"), Size: 11},
			channel,
			&resp.Integer{Val: int64(len(conn.Channels))},
		},
	}
	conn.Write(&res)
}
