package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func unsubscribe(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) != 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'unsubscribe' command"),
		}
		conn.Write(&msg)
		return
	}

	channel, ok := args.Val[1].(*resp.BulkString)
	if !ok {
		msg := resp.SimpleError{
			Val: []byte("wrong data type argument for 'unsubscribe' command"),
		}
		conn.Write(&msg)
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
