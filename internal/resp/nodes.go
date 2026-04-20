package resp

type Message interface {
	ToBytes() []byte
}

type RawMessage []byte

func (r RawMessage) ToBytes() []byte {
	return r
}
