package db

import (
	"time"
)

// This file contains the plumbing needed to take a consistent snapshot of the
// key/value store (for RDB persistence and for the FULLRESYNC part of the
// replication protocol) and to restore one. Every access goes through the shard
// channels, so no locking is needed and no shard is ever touched concurrently.

// SetEntryCommand builds a command that stores a raw entry, used by the RDB load
// path. A zero expiresAt means "no expiry".
func NewSetEntryCommand(key string, value []byte, expiresAt time.Time, done chan<- struct{}) Command {
	return Command{
		key:       key,
		value:     value,
		expiry:    expiresAt,
		doneCh:    done,
		operation: SETENTRY,
	}
}

// NewSnapshotCommand builds a command that asks the shard for a copy of its
// entries. Entries that are already expired are filtered out while the copy is
// taken, the map is then handed over on ch.
func NewSnapshotCommand(ch chan<- map[string]Entry) Command {
	return Command{
		operation: SNAPSHOT,
		snapCh:    ch,
	}
}

// NewFlushCommand builds a command that drops every entry of a shard.
func NewFlushCommand(done chan<- struct{}) Command {
	return Command{
		operation: FLUSH,
		doneCh:    done,
	}
}

// SetEntry stores an entry directly, bypassing the SET command semantics. It is
// used while loading an RDB payload, where the expiry is already absolute.
// Blocks until the owning shard applied the write.
func SetEntry(key string, value []byte, expiresAt time.Time) {
	done := make(chan struct{}, 1)
	GetShardChannel(key) <- NewSetEntryCommand(key, value, expiresAt, done)
	<-done
}

// SnapshotEntries returns a copy of every live entry of the store, keyed by the
// original key. Expired keys are skipped, so the result is what a client would
// currently be able to observe.
func SnapshotEntries() map[string]Entry {
	snapshot := make(map[string]Entry)

	for i := range shards {
		ch := make(chan map[string]Entry, 1)
		shards[i].ch <- NewSnapshotCommand(ch)
		for key, entry := range <-ch {
			snapshot[key] = entry
		}
	}

	return snapshot
}

// FlushStore removes every entry from every shard. It is used before loading an
// RDB payload so that a replica ends up with exactly the contents of the
// snapshot instead of a merge of old and new data.
func FlushStore() {
	for i := range shards {
		done := make(chan struct{}, 1)
		shards[i].ch <- NewFlushCommand(done)
		<-done
	}
}

func handleSetEntryCommand(s *Shard, cmd Command) {
	s.kv[cmd.key] = Entry{Value: cmd.value, ExpiresAt: cmd.expiry}
	if cmd.doneCh != nil {
		cmd.doneCh <- struct{}{}
	}
}

func handleSnapshotCommand(s *Shard, cmd Command) {
	now := time.Now()
	snapshot := make(map[string]Entry, len(s.kv))

	for key, entry := range s.kv {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			// Expired keys are invisible to clients, drop them while we are here
			delete(s.kv, key)
			continue
		}
		snapshot[key] = entry
	}

	cmd.snapCh <- snapshot
}

func handleFlushCommand(s *Shard, cmd Command) {
	s.kv = make(map[string]Entry)
	if cmd.doneCh != nil {
		cmd.doneCh <- struct{}{}
	}
}
