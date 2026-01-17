package resp

type Null struct{}

func (n *Null) ToBytes() []byte {
	return ([]byte("_\r\n"))
}
