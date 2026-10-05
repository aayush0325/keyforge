package rdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Writer incrementally builds an RDB payload. Every value it writes uses the
// classic length encoding of the format, which is understood by all Redis
// versions since 2.6 and therefore keeps the produced files portable.
type Writer struct {
	buf bytes.Buffer
}

// NewWriter returns a writer positioned right after the magic header.
func NewWriter() *Writer {
	w := &Writer{}
	w.buf.Write(Magic)
	return w
}

func (w *Writer) writeByte(b byte) {
	w.buf.WriteByte(b)
}

func (w *Writer) writeBytes(b []byte) {
	w.buf.Write(b)
}

// WriteUint8 writes a raw byte, used for opcodes and type bytes.
func (w *Writer) WriteUint8(v byte) { w.writeByte(v) }

// WriteUint32LE writes a 4 byte little endian integer, this is the byte order
// Redis uses for the fixed size integers of the format (expiry timestamps).
func (w *Writer) WriteUint32LE(v uint32) {
	var scratch [4]byte
	binary.LittleEndian.PutUint32(scratch[:], v)
	w.writeBytes(scratch[:])
}

// WriteUint32BE writes a 4 byte big endian integer, used for lengths.
func (w *Writer) WriteUint32BE(v uint32) {
	var scratch [4]byte
	binary.BigEndian.PutUint32(scratch[:], v)
	w.writeBytes(scratch[:])
}

// WriteUint64BE writes an 8 byte big endian integer, used for lengths.
func (w *Writer) WriteUint64BE(v uint64) {
	var scratch [8]byte
	binary.BigEndian.PutUint64(scratch[:], v)
	w.writeBytes(scratch[:])
}

// WriteUint64LE writes an 8 byte little endian integer.
func (w *Writer) WriteUint64LE(v uint64) {
	var scratch [8]byte
	binary.LittleEndian.PutUint64(scratch[:], v)
	w.writeBytes(scratch[:])
}

// WriteLen writes a length encoded integer.
//
//	00xxxxxx                        6 bit length
//	01xxxxxx xxxxxxxx               14 bit length
//	10000000 <4 byte length>        32 bit length
//	10000001 <8 byte length>        64 bit length
//
// Lengths are stored most significant byte first, which is what Redis does
// (verified against redis 7.4: a 66051 byte value is written as 80 00 01 02 03).
func (w *Writer) WriteLen(n uint64) {
	switch {
	case n < 1<<6:
		w.writeByte(byte(n))
	case n < 1<<14:
		// 01xxxxxx xxxxxxxx: the high bits of the length live in the first byte
		w.writeByte(byte((n>>8)&0x3F) | 0x40)
		w.writeByte(byte(n))
	case n < 1<<32:
		w.writeByte(0x80)
		w.WriteUint32BE(uint32(n))
	default:
		w.writeByte(0x81)
		w.WriteUint64BE(n)
	}
}

// WriteString writes a length encoded byte string.
func (w *Writer) WriteString(s []byte) {
	w.WriteLen(uint64(len(s)))
	w.writeBytes(s)
}

// WriteAux writes an auxiliary metadata field.
func (w *Writer) WriteAux(key string, value []byte) {
	w.writeByte(OpCodeAux)
	w.WriteString([]byte(key))
	w.WriteString(value)
}

// WriteSelectDB selects a keyspace.
func (w *Writer) WriteSelectDB(db uint64) {
	w.writeByte(OpCodeSelectDB)
	w.WriteLen(db)
}

// WriteResizeDB stores the number of keys of the keyspace and how many of them
// carry an expiry.
func (w *Writer) WriteResizeDB(keys, expires uint64) {
	w.writeByte(OpCodeResizeDB)
	w.WriteLen(keys)
	w.WriteLen(expires)
}

// WriteExpireTimeMs writes an absolute expiry as milliseconds since epoch. The
// key type and the key itself have to be written right after it.
func (w *Writer) WriteExpireTimeMs(unixMillis int64) {
	w.writeByte(OpCodeExpireTimeM)
	w.WriteUint64LE(uint64(unixMillis))
}

// WriteExpireTime writes an absolute expiry in seconds since epoch.
func (w *Writer) WriteExpireTime(unixSeconds int64) {
	w.writeByte(OpCodeExpireTime)
	w.WriteUint32LE(uint32(unixSeconds))
}

// WriteTypeAndKey writes the object type byte followed by the key.
func (w *Writer) WriteTypeAndKey(t byte, key string) {
	w.writeByte(t)
	w.WriteString([]byte(key))
}

// WriteEOF terminates the keyspace. No checksum is appended, use AppendChecksum.
func (w *Writer) WriteEOF() { w.writeByte(OpCodeEOF) }

// Bytes returns the payload written so far.
func (w *Writer) Bytes() []byte { return w.buf.Bytes() }

// Len returns the number of bytes written so far.
func (w *Writer) Len() int { return w.buf.Len() }

// Serialize finalizes the payload: EOF opcode plus the 8 byte CRC64.
func (w *Writer) Serialize() []byte {
	w.WriteEOF()
	return AppendChecksum(w.buf.Bytes())
}

// AppendChecksum appends the little endian CRC64 of payload to it.
func AppendChecksum(payload []byte) []byte {
	sum := CRC64(0, payload)
	out := make([]byte, 0, len(payload)+8)
	out = append(out, payload...)
	var scratch [8]byte
	binary.LittleEndian.PutUint64(scratch[:], sum)
	return append(out, scratch[:]...)
}

// Reader parses an RDB payload. All methods return an error instead of
// panicking so that a truncated or corrupted file simply fails to load.
type Reader struct {
	data []byte
	pos  int
}

// NewReader returns a Reader positioned right after the magic header.
func NewReader(data []byte) (*Reader, error) {
	if len(data) < magicLen {
		return nil, fmt.Errorf("rdb: payload too small (%d bytes) to contain a header", len(data))
	}
	if !bytes.HasPrefix(data, Magic[:5]) {
		return nil, fmt.Errorf("rdb: invalid magic %q, file does not look like an RDB dump", data[:magicLen])
	}
	version := data[5:9]
	if bytes.Equal(version, []byte("0000")) || bytes.Equal(version, []byte("0001")) {
		return nil, fmt.Errorf("rdb: unsupported RDB version %q", version)
	}
	return &Reader{data: data, pos: magicLen}, nil
}

// Version returns the RDB version of the payload, e.g. 11.
func (r *Reader) Version() int {
	v := 0
	for _, c := range r.data[5:9] {
		v = v*10 + int(c-'0')
	}
	return v
}

// Len returns the number of bytes left.
func (r *Reader) Len() int { return len(r.data) - r.pos }

// Pos returns the current read offset.
func (r *Reader) Pos() int { return r.pos }

// Rest returns the not yet consumed bytes without consuming them.
func (r *Reader) Rest() []byte { return r.data[r.pos:] }

// Next returns the next byte without consuming it.
func (r *Reader) Next() (byte, bool) {
	if r.pos >= len(r.data) {
		return 0, false
	}
	return r.data[r.pos], true
}

// ReadByte consumes and returns one byte.
func (r *Reader) ReadByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("rdb: unexpected end of payload at offset %d", r.pos)
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

// ReadUint32LE reads a 4 byte little endian integer.
func (r *Reader) ReadUint32LE() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, fmt.Errorf("rdb: unexpected end of payload while reading a 4 byte integer")
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v, nil
}

// ReadUint64LE reads an 8 byte little endian integer.
func (r *Reader) ReadUint64LE() (uint64, error) {
	if r.pos+8 > len(r.data) {
		return 0, fmt.Errorf("rdb: unexpected end of payload while reading an 8 byte integer")
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v, nil
}

// ReadLen reads a length encoded integer.
//
//	00xxxxxx                        6 bit length
//	01xxxxxx xxxxxxxx               14 bit length
//	10000000 <4 byte length>        32 bit length, most significant byte first
//	10000001 <8 byte length>        64 bit length, most significant byte first
//	11xxxxxx                        special integer encodings
//
// A length byte in the 0x82-0xBF range belongs to an LZF compressed value,
// which keyforge does not decompress: ErrCompressed is returned for those.
func (r *Reader) ReadLen() (uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	switch b & 0xC0 {
	case 0x00: // 6 bit length
		return uint64(b & 0x3F), nil
	case 0x40: // 14 bit length
		next, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		return uint64(b&0x3F)<<8 | uint64(next), nil
	}

	if b&0xC0 == 0x80 {
		if b != 0x80 && b != 0x81 {
			// 0x82-0xBF only ever introduces an LZF compressed string
			return 0, ErrCompressed
		}

		n := 4
		if b == 0x81 {
			n = 8
		}
		if r.pos+n > len(r.data) {
			return 0, fmt.Errorf("rdb: truncated %d byte length", n)
		}

		var value uint64
		for i := 0; i < n; i++ {
			value = value<<8 | uint64(r.data[r.pos+i])
		}
		r.pos += n

		return value, nil
	}

	// Special integer encodings, they are written as an int rather than a
	// string length.
	switch b {
	case 0xC0:
		return 0, nil
	case 0xD0:
		return 16384, nil
	case 0xF0:
		v, err := r.ReadUint32LE()
		return uint64(v), err
	case 0xF1:
		return r.ReadUint64LE()
	}

	if b >= 0xC1 && b <= 0xCF { // 8 bit
		v, err := r.ReadByte()
		return uint64(v), err
	}

	if b >= 0xD1 && b <= 0xD3 { // 16, 32 and 64 bit
		n := 1 << (b - 0xD0)
		if r.pos+int(n) > len(r.data) {
			return 0, fmt.Errorf("rdb: truncated integer")
		}
		v := uint64(0)
		for i := 0; i < int(n); i++ {
			v |= uint64(r.data[r.pos+i]) << (8 * uint(i))
		}
		r.pos += int(n)
		return v, nil
	}

	if b >= 0xE0 && b <= 0xEF { // 12, 16, 24, 28 bit
		n := int(b & 0x0F)
		if r.pos+n > len(r.data) {
			return 0, fmt.Errorf("rdb: truncated integer")
		}
		v := uint64(0)
		for i := 0; i < n; i++ {
			v |= uint64(r.data[r.pos+i]) << (8 * uint(i))
		}
		r.pos += n
		return v, nil
	}

	return 0, fmt.Errorf("rdb: unknown length encoding 0x%02x", b)
}

// ReadString reads a length encoded byte string.
func (r *Reader) ReadString() ([]byte, error) {
	n, err := r.ReadLen()
	if err != nil {
		return nil, err
	}
	if r.pos+int(n) > len(r.data) {
		return nil, fmt.Errorf("rdb: truncated string, need %d bytes but only %d are left", n, r.Len())
	}
	s := r.data[r.pos : r.pos+int(n)]
	r.pos += int(n)
	return s, nil
}
