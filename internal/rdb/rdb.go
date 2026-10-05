// Package rdb implements reading and writing of the RDB (REdis Database
// Backup) file format used by Redis for point in time snapshots.
//
// The format is the classic one (RDB version 11 and below, i.e. every Redis
// release up to and including the 7.x line understands it):
//
//	"REDIS0011"                  magic + version
//	<RDB_OPCODE_AUX> key value   auxiliary metadata
//	<RDB_OPCODE_SELECTDB> n      keyspace selector (keyforge only uses db 0)
//	<RDB_OPCODE_RESIZEDB> n m    number of keys and of keys with an expiry
//	[<RDB_OPCODE_EXPIRETIME_MS> ts] <type> <key> <value>
//	...
//	<RDB_OPCODE_EOF>             end of the keyspace
//	<8 byte CRC64 of everything above>
//
// Strings and lists use the standard object types, so a stock Redis can load the
// result of a keyforge dump (verified against redis 7.4 with redis-check-rdb).
//
// Streams are stored with keyforge's own type byte (see TypeStreamKeyforge).
// Redis encodes streams as listpacks inside a serialized radix tree and that
// layout is not reproduced here; 0x1A is not assigned to any object type, so a
// stock Redis refuses the file with a clear "Invalid object type" instead of
// misreading it, while keyforge always reads its own streams back.
package rdb

import "errors"

// Magic header written in front of every payload.
var Magic = []byte("REDIS0011")

// ErrCompressed is returned by (*Reader).ReadLen when the value it read was an
// LZF compressed string. The compressed payload has already been consumed, the
// returned length is the length of the uncompressed value.
var ErrCompressed = errors.New("rdb: value is LZF compressed")

// length of the magic string
const magicLen = 9

// RDB opcodes (see rdb.h in Redis)
const (
	OpCodeEOF         byte = 0xFF
	OpCodeSelectDB    byte = 0xFE
	OpCodeExpireTime  byte = 0xFD // 4 byte, seconds since epoch
	OpCodeExpireTimeM byte = 0xFC // 8 byte, milliseconds since epoch
	OpCodeResizeDB    byte = 0xFB
	OpCodeAux         byte = 0xFA
)

// Object type bytes (see rdb.h in Redis)
const (
	TypeString         byte = 0x00
	TypeList           byte = 0x01 // plain list: <count> then <count> elements
	TypeStreamKeyforge byte = 0x1A // keyforge's own stream encoding
)

// EmptyRDB is a valid, empty RDB payload (magic + EOF + checksum). It is handy
// for tests and as a last resort fallback.
var EmptyRDB = buildEmptyRDB()

func buildEmptyRDB() []byte {
	payload := make([]byte, 0, len(Magic)+5+8)
	payload = append(payload, Magic...)
	payload = append(payload, OpCodeEOF)
	payload = AppendChecksum(payload)
	return payload
}
