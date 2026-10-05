package rdb

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/streams"
)

// Load parses an RDB payload and replaces the content of this instance with it.
//
// The dataset is parsed into memory first and only swapped in once the whole
// payload has been read successfully, so a truncated or corrupt file can never
// leave the server with a half loaded dataset. Callers have to hold the dataset
// lock (see LockDataset), otherwise a write could interleave with the load.
func Load(payload []byte) error {
	dataset, err := Parse(payload)
	if err != nil {
		return err
	}

	Apply(dataset)
	return nil
}

// Apply replaces the content of every store with the given dataset. Callers have
// to hold the dataset lock.
func Apply(dataset *Dataset) {
	db.FlushStore()
	db.FlushLists()
	streams.FlushStreams()

	now := time.Now()

	for key, entry := range dataset.Strings {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			// Expired while the dump was on its way, drop it like Redis does
			continue
		}
		db.SetEntry(key, entry.Value, entry.ExpiresAt)
	}

	for key, elements := range dataset.Lists {
		if len(elements) == 0 {
			continue
		}
		decoded := make([]string, 0, len(elements))
		for _, element := range elements {
			decoded = append(decoded, string(element))
		}
		db.ResetList(key, decoded)
	}

	for key, entries := range dataset.Streams {
		if len(entries) == 0 {
			continue
		}
		streams.RestoreStream(key, entries)
	}
}

// Parse decodes an RDB payload into a Dataset without touching any store.
func Parse(payload []byte) (*Dataset, error) {
	reader, err := NewReader(payload)
	if err != nil {
		return nil, err
	}

	dataset := NewDataset()
	var expiresAt time.Time

	for {
		op, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		switch op {
		case OpCodeEOF:
			return dataset, verifyChecksum(reader, payload)

		case OpCodeAux:
			key, err := reader.ReadString()
			if err != nil {
				return nil, fmt.Errorf("rdb: failed reading aux key: %w", err)
			}
			if _, err := reader.ReadString(); err != nil {
				return nil, fmt.Errorf("rdb: failed reading aux value of %q: %w", key, err)
			}

		case OpCodeSelectDB:
			dbNum, err := reader.ReadLen()
			if err != nil {
				return nil, fmt.Errorf("rdb: failed reading db number: %w", err)
			}
			if dbNum != 0 {
				return nil, fmt.Errorf("rdb: unsupported keyspace %d, keyforge only serves database 0", dbNum)
			}

		case OpCodeResizeDB:
			// The header of the keyspace, the key count is re-derived below
			if _, err := reader.ReadLen(); err != nil {
				return nil, fmt.Errorf("rdb: failed reading key count: %w", err)
			}
			if _, err := reader.ReadLen(); err != nil {
				return nil, fmt.Errorf("rdb: failed reading expiry count: %w", err)
			}

		case OpCodeExpireTime:
			seconds, err := reader.ReadUint32LE()
			if err != nil {
				return nil, fmt.Errorf("rdb: failed reading expiry: %w", err)
			}
			expiresAt = time.Unix(int64(seconds), 0)

		case OpCodeExpireTimeM:
			millis, err := reader.ReadUint64LE()
			if err != nil {
				return nil, fmt.Errorf("rdb: failed reading expiry: %w", err)
			}
			expiresAt = time.UnixMilli(int64(millis))

		case TypeString, TypeList, TypeStreamKeyforge:
			if err := parseValue(reader, dataset, op, expiresAt); err != nil {
				return nil, err
			}
			expiresAt = time.Time{}

		default:
			return nil, fmt.Errorf("rdb: unsupported type/opcode 0x%02x at offset %d", op, reader.Pos()-1)
		}
	}
}

func parseValue(reader *Reader, dataset *Dataset, op byte, expiresAt time.Time) error {
	key, err := reader.ReadString()
	if err != nil {
		return fmt.Errorf("rdb: failed reading key of type 0x%02x: %w", op, err)
	}
	keyStr := string(key)

	switch op {
	case TypeString:
		value, err := reader.ReadString()
		if err != nil {
			return fmt.Errorf("rdb: failed reading value of %q: %w", keyStr, err)
		}
		entry := StringEntry{Value: value}
		if !expiresAt.IsZero() {
			entry.ExpiresAt = expiresAt
		}
		dataset.Strings[keyStr] = entry

	case TypeList:
		count, err := reader.ReadLen()
		if err != nil {
			return fmt.Errorf("rdb: failed reading length of list %q: %w", keyStr, err)
		}
		elements := make([][]byte, 0, count)
		for i := uint64(0); i < count; i++ {
			element, err := reader.ReadString()
			if err != nil {
				return fmt.Errorf("rdb: failed reading element %d of list %q: %w", i, keyStr, err)
			}
			elements = append(elements, element)
		}
		dataset.Lists[keyStr] = elements

	case TypeStreamKeyforge:
		entries, err := reader.readStreamEntries()
		if err != nil {
			return fmt.Errorf("rdb: failed reading entries of stream %q: %w", keyStr, err)
		}
		dataset.Streams[keyStr] = entries
	}

	return nil
}

func (r *Reader) readStreamEntries() ([]*streams.StreamEntry, error) {
	count, err := r.ReadLen()
	if err != nil {
		return nil, err
	}

	entries := make([]*streams.StreamEntry, 0, count)

	for i := uint64(0); i < count; i++ {
		msRaw, err := r.ReadString()
		if err != nil {
			return nil, err
		}
		seqRaw, err := r.ReadString()
		if err != nil {
			return nil, err
		}

		ms, err := strconv.ParseUint(string(msRaw), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid stream milliseconds %q: %w", msRaw, err)
		}
		seq, err := strconv.ParseUint(string(seqRaw), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid stream sequence %q: %w", seqRaw, err)
		}

		fieldCount, err := r.ReadLen()
		if err != nil {
			return nil, err
		}

		fields := make(map[string]string, fieldCount)
		for f := uint64(0); f < fieldCount; f++ {
			field, err := r.ReadString()
			if err != nil {
				return nil, err
			}
			value, err := r.ReadString()
			if err != nil {
				return nil, err
			}
			fields[string(field)] = string(value)
		}

		entries = append(entries, &streams.StreamEntry{
			ID:    &streams.StreamID{Ms: ms, Seq: seq},
			Entry: fields,
		})
	}

	return entries, nil
}

// verifyChecksum compares the 8 trailing bytes of the payload with the CRC64 of
// everything that precedes them.
func verifyChecksum(reader *Reader, payload []byte) error {
	end := reader.Pos()

	trailing := payload[end:]
	if len(trailing) == 0 {
		log.Printf("rdb: payload has no CRC64 checksum (only RDB versions below 5 do this), continuing")
		return nil
	}
	if len(trailing) != 8 {
		return fmt.Errorf("rdb: expected 8 checksum bytes after the EOF opcode, found %d", len(trailing))
	}

	expected := CRC64(0, payload[:end])
	got := le64(trailing)

	if expected != got {
		return fmt.Errorf("rdb: checksum mismatch, file is corrupt (computed %016x, found %016x)", expected, got)
	}

	return nil
}

func le64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// DumpToDisk serializes the dataset and writes it to path. The caller has to
// hold the dataset lock.
//
// The payload is written to a temporary file which is then renamed, so a crash
// (or a concurrent reader such as a replica doing FULLRESYNC) can never observe
// a partially written dump.
func DumpToDisk(path string) error {
	payload, err := Serialize()
	if err != nil {
		return err
	}

	return WritePayloadToDisk(path, payload)
}

// WritePayloadToDisk writes an already serialized payload to path atomically.
func WritePayloadToDisk(path string, payload []byte) error {
	tmpPath := path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := file.Write(payload); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	log.Printf("rdb: saved dataset to %s (%d bytes)", path, len(payload))
	return nil
}

// LoadFromDisk reads the dump at path (if any) and loads it into the stores.
// It is meant for the startup path, where no other command can run yet, so the
// dataset lock is not taken.
func LoadFromDisk(path string) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to restore, the server starts empty
		}
		return err
	}

	if err := Load(payload); err != nil {
		return fmt.Errorf("rdb: failed loading %s: %w", path, err)
	}

	log.Printf("rdb: loaded dataset from %s (%d bytes)", path, len(payload))
	return nil
}
