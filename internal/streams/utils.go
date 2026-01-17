package streams

import (
	"fmt"
	"strconv"
	"strings"
)

func NewStreamID(s string) (*StreamID, error) {
	// Handle fully auto-generated ID (just "*")
	if s == "*" {
		return &StreamID{
			Ms:      0,
			Seq:     0,
			AutoSeq: true,
			AutoMs:  true,
		}, nil
	}

	arr := strings.Split(s, "-")
	if len(arr) != 2 {
		return nil, fmt.Errorf("Unexpected number of '-' in the stream ID")
	}
	ms, err := strconv.ParseUint(arr[0], 10, 64)
	if err != nil {
		return nil, err
	}

	// Handle auto-generated sequence number
	if arr[1] == "*" {
		return &StreamID{
			Ms:      ms,
			Seq:     0,
			AutoSeq: true,
		}, nil
	}

	seq, err := strconv.ParseUint(arr[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &StreamID{
		Ms:  ms,
		Seq: seq,
	}, nil
}

// NewStreamIDForRange parses a stream ID for XRANGE command.
// If no sequence number is provided:
//   - For start ID (isEnd=false): defaults to 0
//   - For end ID (isEnd=true): defaults to max uint64
//
// Special cases:
//   - "-" represents the minimum ID (0-0)
//   - "+" represents the maximum ID (end of stream)
func NewStreamIDForRange(s string, isEnd bool) (*StreamID, error) {
	// Handle "-" as the minimum ID (beginning of stream)
	if s == "-" {
		return &StreamID{
			Ms:  0,
			Seq: 0,
		}, nil
	}

	// Handle "+" as the maximum ID (end of stream)
	if s == "+" {
		return &StreamID{
			Ms:  ^uint64(0), // max uint64
			Seq: ^uint64(0), // max uint64
		}, nil
	}

	arr := strings.Split(s, "-")
	if len(arr) == 1 {
		// No sequence number provided
		ms, err := strconv.ParseUint(arr[0], 10, 64)
		if err != nil {
			return nil, err
		}
		seq := uint64(0)
		if isEnd {
			seq = ^uint64(0) // max uint64
		}
		return &StreamID{
			Ms:  ms,
			Seq: seq,
		}, nil
	}

	if len(arr) != 2 {
		return nil, fmt.Errorf("Unexpected number of '-' in the stream ID")
	}

	ms, err := strconv.ParseUint(arr[0], 10, 64)
	if err != nil {
		return nil, err
	}

	seq, err := strconv.ParseUint(arr[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &StreamID{
		Ms:  ms,
		Seq: seq,
	}, nil
}

func (sid *StreamID) String() string {
	return fmt.Sprintf("%d-%d", sid.Ms, sid.Seq)
}

func newRaxNode(isEnd bool, prefix []byte, entry *StreamEntry) *RaxNode {
	return &RaxNode{
		Entry:        entry,
		IsEndOfEntry: isEnd,
		Prefix:       prefix,
		Children:     make(map[byte]*RaxNode),
	}
}

func newRadixTrie(prefix []byte, entry *StreamEntry) *Rax {
	return &Rax{
		Root: newRaxNode(true, prefix, entry),
	}
}

func NewStream(entry *StreamEntry) *Stream {
	return &Stream{
		LastEntry: entry,
		Radix:     newRadixTrie([]byte(entry.ID.String()), entry),
	}
}

func NewEmptyStream() *Stream {
	return &Stream{
		Radix: &Rax{
			Root: &RaxNode{
				Children: make(map[byte]*RaxNode),
			},
		},
	}
}
