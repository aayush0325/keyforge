package config

import "sync/atomic"

var IsReplica bool
var DebugMode bool
var ReplID string

// ShuttingDown is set once the process received a termination signal.
var ShuttingDown atomic.Bool

// Offset is the replication offset of this instance, it is only touched through
// the accessors below so that concurrent command goroutines do not race on it.
var Offset uint64

var LastConfirmedOffset uint64

// Where the RDB file lives. Both are overridden by the -dir / -dbfilename flags
// and can be changed at runtime with CONFIG SET.
var Dir = "."

// DBFilename is the name of the RDB file inside Dir.
var DBFilename = "dump.rdb"

// GetOffset returns the current replication offset.
func GetOffset() uint64 { return atomic.LoadUint64(&Offset) }

// AddOffset advances the replication offset by n bytes, every byte sent to (or
// received from) a replica moves it forward.
func AddOffset(n uint64) { atomic.AddUint64(&Offset, n) }

// SetOffset sets the replication offset, used after a full resynchronization.
func SetOffset(n uint64) { atomic.StoreUint64(&Offset, n) }

// GetLastConfirmedOffset returns the offset the replicas confirmed to have
// applied.
func GetLastConfirmedOffset() uint64 { return atomic.LoadUint64(&LastConfirmedOffset) }

// SetLastConfirmedOffset records the offset confirmed by the replicas.
func SetLastConfirmedOffset(n uint64) { atomic.StoreUint64(&LastConfirmedOffset, n) }
