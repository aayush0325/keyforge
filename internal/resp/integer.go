package resp

import "strconv"

type Integer struct {
	Val int64
}

func (i *Integer) ToBytes() []byte {
	res := ([]byte)(":")
	res = append(res, ([]byte)(strconv.FormatInt(i.Val, 10))...)
	res = append(res, ([]byte)("\r\n")...)
	return res
}
