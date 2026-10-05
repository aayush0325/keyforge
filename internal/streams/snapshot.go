package streams

// This file contains the plumbing to dump and restore streams, it is used by
// the RDB persistence path as well as by FULLRESYNC on the replica side.

// SnapshotStreams returns a copy of every stream with its entries ordered by ID.
// The returned entries are copies, so the caller can serialize them without
// holding the global lock.
func SnapshotStreams() map[string][]*StreamEntry {
	Global.Mu.Lock()
	defer Global.Mu.Unlock()

	snapshot := make(map[string][]*StreamEntry, len(Global.KV))

	for key, stream := range Global.KV {
		if stream == nil || stream.Radix == nil {
			continue
		}

		entries := stream.Range(&StreamID{Ms: 0, Seq: 0}, &StreamID{Ms: ^uint64(0), Seq: ^uint64(0)})
		copied := make([]*StreamEntry, 0, len(entries))

		for _, entry := range entries {
			fields := make(map[string]string, len(entry.Entry))
			for field, value := range entry.Entry {
				fields[field] = value
			}
			copied = append(copied, &StreamEntry{
				ID:    &StreamID{Ms: entry.ID.Ms, Seq: entry.ID.Seq},
				Entry: fields,
			})
		}

		snapshot[key] = copied
	}

	return snapshot
}

// RestoreStream recreates a stream from entries ordered by ID. Entries that are
// already present are replaced, which is what the RDB load path wants after
// flushing the store.
func RestoreStream(key string, entries []*StreamEntry) {
	Global.Mu.Lock()
	defer Global.Mu.Unlock()

	stream, exists := Global.KV[key]
	if !exists {
		stream = NewEmptyStream()
		Global.KV[key] = stream
	}

	for _, entry := range entries {
		// Entries are sorted by ID, so the last insert ends up being the last
		// entry of the stream, which is what XADD relies on.
		stream.Insert(entry, entry.ID.InternalKey())
	}
}

// FlushStreams drops every stream, used before loading an RDB payload so that a
// replica holds exactly the contents of the snapshot.
func FlushStreams() {
	Global.Mu.Lock()
	defer Global.Mu.Unlock()

	Global.KV = make(map[string]*Stream)
}
