package resp

import "strconv"

type BulkString struct {
	Str  []byte
	Size int
}

const NULLBULKSTRING = "$-1\r\n"

func (b *BulkString) ToBytes() []byte {
	res := []byte("$")
	res = append(res, ([]byte)(strconv.FormatInt(int64(b.Size), 10))...)
	res = append(res, ([]byte)("\r\n")...)
	if b.Str == nil {
		// handle null bulk string
		return res
	}
	res = append(res, b.Str...)
	res = append(res, ([]byte)("\r\n")...)
	return res
}
