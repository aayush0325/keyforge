package streams

import "sync"

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

	// Collect all entries using DFS, iterating children in sorted byte order
	var collectAll func(node *RaxNode)
	collectAll = func(node *RaxNode) {
		if node == nil {
			return
		}
		if node.IsEndOfEntry && node.Entry != nil {
			// Check if the entry is within the range
			if node.Entry.ID.Compare(start) >= 0 && node.Entry.ID.Compare(end) <= 0 {
				result = append(result, node.Entry)
			}
		}
		// Iterate children in sorted byte order (0-255)
		for i := 0; i <= 255; i++ {
			if child, exists := node.Children[byte(i)]; exists {
				collectAll(child)
			}
		}
	}

	collectAll(s.Radix.Root)

	// Sort by ID (since different branches may have mixed ordering)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].ID.Compare(result[j].ID) > 0 {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

type GlobalInstance struct {
	Mu sync.Mutex
	KV map[string]*Stream
}
