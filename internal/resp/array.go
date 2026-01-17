package resp

import "strconv"

type Array struct {
	Val []Message
}

func (a *Array) ToBytes() []byte {
	res := ([]byte)("*")
	res = append(res, ([]byte)(strconv.Itoa(len(a.Val)))...)
	res = append(res, ([]byte)("\r\n")...)
	for _, el := range a.Val {
		res = append(res, el.ToBytes()...)
	}
	return res
}
