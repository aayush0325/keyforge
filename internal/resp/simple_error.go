package resp

type SimpleError struct {
	Val []byte
}

func (se *SimpleError) ToBytes() []byte {
	res := ([]byte)("-")
	res = append(res, se.Val...)
	res = append(res, ([]byte)("\r\n")...)
	return res
}
