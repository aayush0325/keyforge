package commands

import (
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/resp"
)

func subscribe(args *resp.Array, conn *pubsub.Connection) {
	if len(args.Val) < 2 {
		msg := resp.SimpleError{
			Val: []byte("wrong number of arguments for 'subscribe' command"),
		}
		conn.Write(&msg)
		return
	}
	channels := make([][]byte, 0)

	for i := 1; i < len(args.Val); i++ {
		channel, ok := args.Val[i].(*resp.BulkString)
		if !ok {
			msg := resp.SimpleError{
				Val: []byte("wrong data type argument for 'echo' command"),
			}
			conn.Write(&msg)
			return
		}
		channels = append(channels, channel.Str)
		conn.Channels[string(channel.Str)] = struct{}{} // update the connection to channel mapping
	}

	pubsub.Instance.Mu.Lock()
	for _, ch := range channels {
		_, ok := pubsub.Instance.ChannelToClient[string(ch)]
		// update the channel to connection mapping
		if !ok {
			pubsub.Instance.ChannelToClient[string(ch)] = map[*pubsub.Connection]struct{}{conn: {}}
		} else {
			pubsub.Instance.ChannelToClient[string(ch)][conn] = struct{}{}
		}
		res := resp.Array{
			Val: []resp.Message{
				&resp.BulkString{Str: []byte("subscribe"), Size: 9},
				&resp.BulkString{Str: ch, Size: len(ch)},
				&resp.Integer{Val: int64(len(conn.Channels))},
			},
		}
		conn.Write(&res)
	}
	pubsub.Instance.Mu.Unlock()
}
