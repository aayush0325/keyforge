package rdb

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/streams"
)

// KeyforgeVersion is advertised in the auxiliary "redis-ver" field.
var KeyforgeVersion = []byte("keyforge-1.0")

// StringEntry is a string key with its optional expiry.
type StringEntry struct {
	Value     []byte
	ExpiresAt time.Time
}

// Dataset is an in memory representation of everything an RDB payload holds.
type Dataset struct {
	Strings map[string]StringEntry
	Lists   map[string][][]byte
	Streams map[string][]*streams.StreamEntry
}

// NewDataset returns an empty dataset.
func NewDataset() *Dataset {
	return &Dataset{
		Strings: make(map[string]StringEntry),
		Lists:   make(map[string][][]byte),
		Streams: make(map[string][]*streams.StreamEntry),
	}
}

// Len returns the total number of keys held by the dataset.
func (d *Dataset) Len() int {
	return len(d.Strings) + len(d.Lists) + len(d.Streams)
}

// datasetMu serializes write commands against a dump of the dataset.
//
// Write commands take it exclusively (rdb.LockDataset) from before they are
// executed until they were propagated to the replicas, while a dump takes it
// exclusively as well. That way a replica can never receive a write twice:
// either the write made it into the snapshot, or it is propagated afterwards.
//
// The lock is not reentrant: taking it twice from the same goroutine deadlocks,
// callers that may already hold it (EXEC of a transaction) have to check first.
var datasetMu sync.RWMutex

// LockDataset blocks until no other dataset operation is in flight. Write
// commands and dumps both take it exclusively.
func LockDataset() { datasetMu.Lock() }

// UnlockDataset releases the exclusive dataset lock.
func UnlockDataset() { datasetMu.Unlock() }

// Serialize returns an RDB payload holding the whole dataset of this instance.
//
// Callers have to hold the dataset lock exclusively (see LockDataset) so that no
// write command interleaves with the snapshot, otherwise the payload could
// contain a write that is propagated to the replicas afterwards.
func Serialize() ([]byte, error) {
	return SerializeDataset(snapshotLiveDataset())
}

// snapshotLiveDataset copies the current content of every store. Callers have to
// hold the dataset lock.
func snapshotLiveDataset() *Dataset {
	dataset := NewDataset()

	for key, entry := range db.SnapshotEntries() {
		value := make([]byte, len(entry.Value))
		copy(value, entry.Value)
		dataset.Strings[key] = StringEntry{Value: value, ExpiresAt: entry.ExpiresAt}
	}

	for key, elements := range db.SnapshotLists() {
		list := make([][]byte, 0, len(elements))
		for _, element := range elements {
			list = append(list, []byte(element))
		}
		dataset.Lists[key] = list
	}

	for key, entries := range streams.SnapshotStreams() {
		dataset.Streams[key] = entries
	}

	return dataset
}

// SerializeDataset encodes a dataset as an RDB payload.
func SerializeDataset(dataset *Dataset) ([]byte, error) {
	writer := NewWriter()

	now := time.Now()

	writer.WriteAux("redis-ver", KeyforgeVersion)
	writer.WriteAux("redis-bits", []byte("64"))
	writer.WriteAux("ctime", []byte(fmt.Sprintf("%d", now.Unix())))
	writer.WriteAux("used-mem", []byte("0"))

	writer.WriteSelectDB(0)

	// Work out which keys are actually written before emitting the RESIZEDB
	// header, it has to state the number of keys and of keys with an expiry up
	// front. A key that exists in more than one store is only written once, the
	// string store wins as it is the primary keyspace, and empty lists or streams
	// are skipped because they are indistinguishable from a missing key.
	writtenStrings := make(map[string]struct{}, len(dataset.Strings))
	writtenLists := make(map[string]struct{}, len(dataset.Lists))
	writtenStreams := make(map[string]struct{}, len(dataset.Streams))
	expiredKeys := 0

	for key, entry := range dataset.Strings {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			continue // already expired, a client would not see it either
		}
		writtenStrings[key] = struct{}{}
		if !entry.ExpiresAt.IsZero() {
			expiredKeys++
		}
	}

	for key, elements := range dataset.Lists {
		if len(elements) == 0 || isAlreadyWritten(writtenStrings, key) {
			continue
		}
		writtenLists[key] = struct{}{}
	}

	for key, entries := range dataset.Streams {
		if len(entries) == 0 || isAlreadyWritten(writtenStrings, key) || isAlreadyWritten(writtenLists, key) {
			continue
		}
		writtenStreams[key] = struct{}{}
	}

	totalKeys := len(writtenStrings) + len(writtenLists) + len(writtenStreams)
	writer.WriteResizeDB(uint64(totalKeys), uint64(expiredKeys))

	// The keys are written in a stable order, so dumping the same data twice
	// yields the very same bytes.
	for _, key := range sortedKeys(writtenStrings) {
		entry := dataset.Strings[key]

		if !entry.ExpiresAt.IsZero() {
			writer.WriteExpireTimeMs(entry.ExpiresAt.UnixMilli())
		}

		writer.WriteTypeAndKey(TypeString, key)
		writer.WriteString(entry.Value)
	}

	for _, key := range sortedKeys(writtenLists) {
		elements := dataset.Lists[key]

		writer.WriteTypeAndKey(TypeList, key)
		writer.WriteLen(uint64(len(elements)))
		for _, element := range elements {
			writer.WriteString(element)
		}
	}

	for _, key := range sortedKeys(writtenStreams) {
		writer.WriteTypeAndKey(TypeStreamKeyforge, key)
		writer.writeStreamEntries(dataset.Streams[key])
	}

	return writer.Serialize(), nil
}

// isAlreadyWritten reports whether the key was claimed by another store, which
// happens when the same name is used for a string and a list or stream.
func isAlreadyWritten(written map[string]struct{}, key string) bool {
	if _, ok := written[key]; ok {
		log.Printf("rdb: key %q exists in more than one store, the string value is kept", key)
		return true
	}
	return false
}

// writeStreamEntries writes the entries of a stream.
//
// Layout (keyforge specific, see the package documentation):
//
//	<number of entries>
//	<milliseconds> <sequence> <number of fields> (<field> <value>)*
//
// repeated once per entry, fields sorted by name so that dumping the same data
// twice yields byte identical payloads.
func (w *Writer) writeStreamEntries(entries []*streams.StreamEntry) {
	w.WriteLen(uint64(len(entries)))

	for _, entry := range entries {
		w.WriteString([]byte(fmt.Sprintf("%d", entry.ID.Ms)))
		w.WriteString([]byte(fmt.Sprintf("%d", entry.ID.Seq)))
		w.WriteLen(uint64(len(entry.Entry)))

		fields := make([]string, 0, len(entry.Entry))
		for field := range entry.Entry {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		for _, field := range fields {
			w.WriteString([]byte(field))
			w.WriteString([]byte(entry.Entry[field]))
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
