package rdb

// CRC64 implementation used by Redis for RDB payload checksums.
//
// It is the "Jones" polynomial (0xad93d23594c935a9) applied in reflected
// (LSB first) fashion with an initial value of 0 and no final xor, which is
// exactly what Redis' crc64.c implements. The tables below are the slicing-by-8
// variant so that dumping a large dataset stays cheap.

const crc64Poly = uint64(0xad93d23594c935a9)

// reversed polynomial, used by the reflected bit loop below
const crc64PolyReflected = uint64(0x95ac9329ac4bc9b5)

var crc64Tables = buildCRCTables()

func buildCRCTables() [8][256]uint64 {
	var tables [8][256]uint64

	for i := 0; i < 256; i++ {
		crc := uint64(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ crc64PolyReflected
			} else {
				crc >>= 1
			}
		}
		tables[0][i] = crc
	}

	// Slicing-by-8 tables: table[i] folds the byte into the CRC shifted by i
	// bytes, allowing 8 bytes to be processed per iteration.
	for i := 1; i < 8; i++ {
		for j := 0; j < 256; j++ {
			crc := tables[i-1][j]
			tables[i][j] = (crc >> 8) ^ tables[0][crc&0xFF]
		}
	}

	return tables
}

// CRC64 returns the Redis compatible CRC64 checksum of b, seeded with prev so
// that a payload can be checksummed incrementally.
func CRC64(prev uint64, b []byte) uint64 {
	crc := prev

	for len(b) >= 8 {
		crc ^= uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
		crc = crc64Tables[7][crc&0xFF] ^
			crc64Tables[6][(crc>>8)&0xFF] ^
			crc64Tables[5][(crc>>16)&0xFF] ^
			crc64Tables[4][(crc>>24)&0xFF] ^
			crc64Tables[3][(crc>>32)&0xFF] ^
			crc64Tables[2][(crc>>40)&0xFF] ^
			crc64Tables[1][(crc>>48)&0xFF] ^
			crc64Tables[0][(crc>>56)&0xFF]
		b = b[8:]
	}

	for _, c := range b {
		crc = crc64Tables[0][byte(crc)^c] ^ (crc >> 8)
	}

	return crc
}

// bitwiseCRC64 is the straightforward definition kept around for tests, it is
// the reference implementation the fast table driven version above is checked
// against.
func bitwiseCRC64(crc uint64, b []byte) uint64 {
	for _, c := range b {
		crc ^= uint64(c)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ crc64PolyReflected
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
