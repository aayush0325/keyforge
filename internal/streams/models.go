package streams

import (
	"fmt"
	"sync"
)

type StreamEntry struct {
	ID    *StreamID
	Entry map[string]string
}

type StreamID struct {
	Ms      uint64
	Seq     uint64
	AutoSeq bool // true when sequence number should be auto-generated
	AutoMs  bool // true when entire ID (time + sequence) should be auto-generated
}

// Compare compares two StreamIDs
// Returns -1 if this < other, 0 if equal, 1 if this > other
func (sid *StreamID) Compare(other *StreamID) int {
	if sid.Ms < other.Ms {
		return -1
	}
	if sid.Ms > other.Ms {
		return 1
	}
	// Ms are equal, compare Seq
	if sid.Seq < other.Seq {
		return -1
	}
	if sid.Seq > other.Seq {
		return 1
	}
	return 0
}

// InternalKey returns a zero-padded string representation for lexicographical ordering in the trie
func (sid *StreamID) InternalKey() []byte {
	return []byte(fmt.Sprintf("%020d-%020d", sid.Ms, sid.Seq))
}

// IsZero returns true if the StreamID is 0-0
func (sid *StreamID) IsZero() bool {
	return sid.Ms == 0 && sid.Seq == 0
}

type BlockingListener struct {
	Channel   chan struct{}
	WaitingID *StreamID
}

type Stream struct {
	LastEntry         *StreamEntry
	Radix             *Rax
	BlockingListeners []*BlockingListener
}

func (s *Stream) Insert(se *StreamEntry, prefix []byte) {
	s.Radix.Insert(prefix, se)
	s.LastEntry = se
}

// Range returns all entries with IDs in the range [start, end] (inclusive)
func (s *Stream) Range(start, end *StreamID) []*StreamEntry {
	var result []*StreamEntry

	startKey := start.InternalKey()
	node := s.Radix.SeekGE(startKey)
	for node != nil {
		// Check if the entry is within the range
		if node.Entry.ID.Compare(end) > 0 {
			break
		}
		result = append(result, node.Entry)

		// Get next entry
		node = s.Radix.Successor(node.Entry.ID.InternalKey())
	}

	return result
}

type GlobalInstance struct {
	Mu sync.Mutex
	KV map[string]*Stream
}
