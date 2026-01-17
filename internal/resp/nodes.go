package resp

type Message interface {
	ToBytes() []byte
}
