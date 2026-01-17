package resp

type SimpleString struct {
	Val []byte
}

func (s *SimpleString) ToBytes() []byte {
	res := ([]byte)("+")
	res = append(res, s.Val...)
	res = append(res, []byte("\r\n")...)
	return res
}
