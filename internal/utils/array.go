package utils

import "github.com/codecrafters-io/redis-starter-go/internal/resp"

// Takes an array of strings and returns a RESP array of bulk strings
func GetRespArrayBulkString(arr []string) resp.Array {
	res := resp.Array{Val: make([]resp.Message, 0)}
	for _, el := range arr {
		bulk := &resp.BulkString{Str: []byte(el), Size: len(el)}
		res.Val = append(res.Val, bulk)
	}
	return res
}
