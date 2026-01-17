package resp

type Boolean struct {
	Val bool
}

func (b *Boolean) ToBytes() []byte {
	if b.Val {
		return []byte("#t\r\n")
	}
	return []byte("#f\r\n")
}
