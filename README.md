# Keyforge - A Redis Implementation in Go

Keyforge is a feature-rich Redis server implementation written in Go. It provides a subset of Redis commands optimized for performance and simplicity while maintaining protocol compatibility with Redis clients.

## Features

- **RESP Protocol Support**: Full Redis Serialization Protocol (RESP) implementation for client-server communication
- **Data Structures**: Support for Strings, Lists, and Streams
- **Pub/Sub Messaging**: Publish-Subscribe pattern implementation for real-time messaging
- **Persistence**: RDB snapshots of strings (with TTL), lists and streams, loaded on startup and written on shutdown
- **Replication**: Full resynchronization with `PSYNC`/`REPLCONF`, command streaming and `WAIT` acknowledgements
- **Connection Handling**: Multi-threaded concurrent connection handling
- **Debug Mode**: Optional debug logging for command execution

## Getting Started

### Prerequisites

- Go 1.25.0 or later

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd keyforge
```

2. Build the server:
```bash
go build -o keyforge ./app
```

3. Run the server:
```bash
./keyforge
```

The server will start and listen on `localhost:6379` (default Redis port).

### Command line flags

| Flag | Default | Description |
| --- | --- | --- |
| `-port` | `6379` | Port to listen on |
| `-replicaof` | *(empty)* | Act as a replica of `<host> <port>` |
| `-dir` | `.` | Directory holding the RDB file |
| `-dbfilename` | `dump.rdb` | Name of the RDB file inside `-dir` |
| `-nosave` | `false` | Do not write an RDB file on shutdown |
| `-debug` | `false` | Log every command |

### Debug Mode

Run with debug logging enabled:
```bash
./keyforge -debug
```

This will log all incoming commands to the console for debugging purposes.

## Persistence

Keyforge stores its dataset as an RDB (REdis Database Backup) file, the same
format Redis uses for its snapshots. The file is loaded on startup and written on
`SAVE`, `BGSAVE` and when the process is asked to terminate.

```bash
./keyforge -dir /var/lib/keyforge -dbfilename dump.rdb
redis-cli -p 6379 set greeting "hello"
redis-cli -p 6379 save        # writes /var/lib/keyforge/dump.rdb
```

```
redis-cli -p 6379 config get dir dbfilename
redis-cli -p 6379 bgsave      # snapshot in the background
redis-cli -p 6379 lastsave    # unix timestamp of the last successful save
redis-cli -p 6379 debug reload # reload the dump from disk, useful to verify it
```

What is persisted:

| Type | Encoding | Notes |
| --- | --- | --- |
| Strings | `RDB_TYPE_STRING` | Binary safe, expiries are stored with `RDB_OPCODE_EXPIRETIME_MS` |
| Lists | `RDB_TYPE_LIST` | Element order is preserved |
| Streams | keyforge specific type `0x1A` | See the note below |

Payloads are written to a temporary file and renamed into place, so a crash or a
concurrently loading replica never observes a half written dump. Keys that
expired while the snapshot was being taken are dropped, and loading a payload
replaces the whole dataset instead of merging into it.

### Compatibility

The strings, lists and expiry timestamps use the classic RDB encoding, so a dump
written by keyforge can be loaded by a stock Redis (verified against redis 7.4
with `redis-check-rdb` and by loading a 20k key dump with a 30 KB value and a
TTL into a real server). Fixed size integers follow the byte order of Redis:
lengths are big endian, expiry timestamps are little endian, and the payload
ends with the CRC64 checksum Redis uses.

Streams are the exception. Redis encodes them as listpacks stored in a
serialized radix tree, a layout keyforge does not reproduce, so it uses its own
type byte. `0x1A` is not assigned to any object type, which means a stock Redis
stops with a clear `Invalid object type: 26` instead of misreading the entries,
while keyforge itself always reads its own streams back.

Dumps written by keyforge never use LZF compression. A dump produced by a stock
Redis may contain compressed values and sets, those files are rejected with a
`value is LZF compressed` error instead of being partially loaded.

## Replication

A keyforge instance can replicate from another one (and, being RESP compatible,
be inspected with `redis-cli`):

```bash
# master
./keyforge -port 6379 -dir /var/lib/keyforge

# replica: performs the handshake, applies the snapshot and starts streaming
./keyforge -port 6380 -dir /var/lib/keyforge-replica -replicaof "localhost 6379"
```

1. The replica sends `PING`, `REPLCONF listening-port` and `REPLCONF capa`.
2. It sends `PSYNC ? -1` and the master answers `+FULLRESYNC <replid> <offset>`.
3. The master serializes the whole dataset as an RDB payload and sends it as a
   bulk string. Write commands are held back for the duration of the snapshot, so
   a write can never end up both inside the snapshot and on the replication
   stream.
4. The replica parses the payload, verifies its CRC64 and applies it to its own
   stores before it starts serving clients, then continues reading the
   replication stream.
5. Every write command (`SET`, `SETNX`, `DEL`, `INCR`, `LPUSH`, `RPUSH`, `LPOP`,
   `XADD`) is streamed to the attached replicas afterwards.
6. `WAIT <numreplicas> <timeout>` asks the replicas to acknowledge the offset,
   which they answer with `REPLCONF ACK <offset>`.

Both sides count the bytes of the replication stream in `master_repl_offset`, so
`INFO replication` reports the same offset on the master and its replicas.

Because keyforge does not keep a replication backlog, every `PSYNC` triggers a
full resynchronization: there is no partial resynchronization and no handling of
a broken replica yet.

### Manual test

```bash
redis-cli -p 6379 set greeting "hello replication"
redis-cli -p 6380 get greeting          # hello replication, received through FULLRESYNC
redis-cli -p 6379 incr counter
redis-cli -p 6380 get counter           # streamed to the replica
redis-cli -p 6379 wait 1 1000           # 1, the replica acknowledged the write
redis-cli -p 6379 info replication
redis-cli -p 6380 info replication
```

The complete flow is covered by an end to end script that drives two servers
with `redis-cli`:

```bash
tests/test_persistence_and_replication.sh
```

## Supported Commands

### String Commands

#### SET
Set a key-value pair with optional TTL (time-to-live).

**Syntax:**
```
SET key value [EX seconds] [PX milliseconds] [NX | XX]
```

**Examples:**
```
SET mykey "Hello"
SET mykey "World" EX 10
SET mykey "Value" NX  # Only set if key doesn't exist
SET mykey "Value" XX  # Only set if key exists
```

**Return:** Simple string "OK"

---

#### SETNX
Set a key-value pair only if the key does not already exist.

**Syntax:**
```
SETNX key value
```

**Examples:**
```
SETNX mykey "Hello"
```

**Return:** Integer (1 if set, 0 if not set)

---

#### GET
Retrieve the value of a key.

**Syntax:**
```
GET key
```

**Examples:**
```
GET mykey
```

**Return:** Bulk string with the value, or null if key doesn't exist

---

### List Commands

#### LPUSH
Insert one or more values at the head of a list.

**Syntax:**
```
LPUSH key value [value ...]
```

**Examples:**
```
LPUSH mylist "world"
LPUSH mylist "hello"
```

**Return:** Integer representing the length of the list after the operation

---

#### RPUSH
Insert one or more values at the tail of a list.

**Syntax:**
```
RPUSH key value [value ...]
```

**Examples:**
```
RPUSH mylist "one"
RPUSH mylist "two"
```

**Return:** Integer representing the length of the list after the operation

---

#### LPOP
Remove and return the first element of a list.

**Syntax:**
```
LPOP key [count]
```

**Examples:**
```
LPOP mylist
```

**Return:** Bulk string with the popped value, or null if list is empty

---

#### BLPOP
Blocking version of LPOP. Waits for an element to be available.

**Syntax:**
```
BLPOP key [key ...] timeout
```

**Examples:**
```
BLPOP mylist 0
```

**Return:** Array containing the key and the value, or null if timeout expires

---

#### LLEN
Get the length of a list.

**Syntax:**
```
LLEN key
```

**Examples:**
```
LLEN mylist
```

**Return:** Integer representing the number of elements in the list

---

#### LRANGE
Get a range of elements from a list.

**Syntax:**
```
LRANGE key start stop
```

**Examples:**
```
LRANGE mylist 0 -1  # Get all elements
LRANGE mylist 0 2   # Get first 3 elements
```

**Return:** Array of bulk strings representing the elements

---

### Stream Commands

#### XADD
Add an entry to a stream.

**Syntax:**
```
XADD key ID field value [field value ...]
```

**Examples:**
```
XADD mystream * name "Alice" age "30"
XADD mystream 1000-0 field1 "value1"
```

**Return:** Bulk string with the ID of the added entry

---

#### XRANGE
Get a range of entries from a stream.

**Syntax:**
```
XRANGE key start end [COUNT count]
```

**Examples:**
```
XRANGE mystream - +                    # Get all entries
XRANGE mystream 1000-0 2000-0          # Get entries within ID range
XRANGE mystream - + COUNT 10           # Get first 10 entries
```

**Return:** Array of entries (each entry is a pair of [ID, [field, value, ...]])

---

#### XREAD
Read from one or more streams.

**Syntax:**
```
XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] id [id ...]
```

**Examples:**
```
XREAD STREAMS mystream 0            # Read all entries from mystream
XREAD COUNT 2 STREAMS mystream 0    # Read 2 entries
XREAD BLOCK 1000 STREAMS mystream $ # Block for 1000ms, read new entries
```

**Return:** Array of streams with their entries

---

### Key Commands

#### DEL
Delete one or more keys.

**Syntax:**
```
DEL key [key ...]
```

**Examples:**
```
DEL mykey
DEL key1 key2 key3
```

**Return:** Integer representing the number of keys deleted

---

#### EXISTS
Check if one or more keys exist.

**Syntax:**
```
EXISTS key [key ...]
```

**Examples:**
```
EXISTS mykey
EXISTS key1 key2 key3
```

**Return:** Integer representing the number of existing keys

---

#### TYPE
Get the type of a key.

**Syntax:**
```
TYPE key
```

**Examples:**
```
TYPE mystring   # Returns "string"
TYPE mylist     # Returns "list"
TYPE mystream   # Returns "stream"
```

**Return:** Simple string representing the type (string, list, stream, none)

---

### Pub/Sub Commands

#### PUBLISH
Publish a message to a channel.

**Syntax:**
```
PUBLISH channel message
```

**Examples:**
```
PUBLISH mychannel "Hello, World!"
```

**Return:** Integer representing the number of subscribers that received the message

---

#### SUBSCRIBE
Subscribe to one or more channels.

**Syntax:**
```
SUBSCRIBE channel [channel ...]
```

**Examples:**
```
SUBSCRIBE mychannel
SUBSCRIBE channel1 channel2
```

**Return:** Array messages containing the subscription confirmations and messages

---

#### UNSUBSCRIBE
Unsubscribe from one or more channels.

**Syntax:**
```
UNSUBSCRIBE [channel [channel ...]]
```

**Examples:**
```
UNSUBSCRIBE mychannel
UNSUBSCRIBE  # Unsubscribe from all channels
```

**Return:** Array messages containing the unsubscribe confirmations

---

### Connection Commands

#### PING
Test the connection to the server.

**Syntax:**
```
PING [message]
```

**Examples:**
```
PING
PING "Hello"
```

**Return:** Simple string "PONG" or the provided message

---

#### ECHO
Echo the provided message.

**Syntax:**
```
ECHO message
```

**Examples:**
```
ECHO "Hello, Redis!"
```

**Return:** Bulk string with the echoed message

---

#### HELLO
Greeting command (Redis 6.0+ compatibility).

**Syntax:**
```
HELLO [protover]
```

**Examples:**
```
HELLO
HELLO 3
```

**Return:** Array with server information

---

#### CLIENT
Manage client connections.

**Syntax:**
```
CLIENT subcommand [args]
```

**Note:** Partial implementation for Redis-CLI compatibility

---

#### CONFIG
Get or set configuration parameters. `dir` and `dbfilename` decide where `SAVE`
and the FULLRESYNC payload are written, both can be changed at runtime.

**Syntax:**
```
CONFIG GET pattern
CONFIG SET parameter value
```

**Examples:**
```
CONFIG GET dir dbfilename
CONFIG SET dbfilename other.rdb
```

**Return:** Array with configuration values or status

---

#### COMMAND
Get information about Redis commands (Redis-CLI compatibility).

**Syntax:**
```
COMMAND
```

**Return:** Array of command information

---

#### SAVE
Synchronously write the dataset to the RDB file.

**Syntax:**
```
SAVE
```

**Return:** Simple string "OK"

---

#### BGSAVE
Write the dataset to the RDB file in the background while the server keeps
serving clients.

**Syntax:**
```
BGSAVE
```

**Return:** Simple string "Background saving started"

---

#### LASTSAVE
Unix timestamp of the last successful `SAVE`/`BGSAVE`.

**Syntax:**
```
LASTSAVE
```

**Return:** Integer timestamp

---

#### DEBUG RELOAD
Reload the dataset from the RDB file on disk, replacing the current content.

**Syntax:**
```
DEBUG RELOAD
```

**Return:** Simple string "OK"

---

## Data Types

### Strings
Keyforge stores simple key-value pairs where both keys and values are binary-safe strings. Supports TTL (Time-To-Live) for automatic key expiration.

### Lists
Doubly-linked list implementation supporting LPUSH, RPUSH, LPOP, BLPOP, LLEN, and LRANGE operations. Lists are created implicitly when the first element is added.

### Streams
Time-series data structure with entries identified by their timestamp (ID). Each entry contains a set of field-value pairs. Supports efficient range queries and blocking reads.

## Internal Architecture

### Components

**Core Modules:**
- **Parser** (`internal/parser/`): RESP protocol parser for client requests
- **Commands** (`internal/commands/`): Command execution handlers
- **Database** (`internal/db/`): In-memory data storage with shard-based concurrency
- **Pub/Sub** (`internal/pubsub/`): Message broker for publish-subscribe functionality
- **Streams** (`internal/streams/`): Stream data structure implementation with Radix tree support
- **RDB** (`internal/rdb/`): RDB snapshot reader/writer and the dataset dump used by FULLRESYNC
- **Utils** (`internal/utils/`): Helper utilities and data structures
- **RESP** (`internal/resp/`): Redis Serialization Protocol implementation
- **Deque** (`internal/ds/`): Double-ended queue data structure

### Threading Model

Keyforge uses a multi-threaded architecture:
- Each client connection is handled in a separate goroutine
- Database operations use sharded locks for high concurrency
- Pub/Sub uses global state with connection locks for message delivery
- All operations are thread-safe

### Dataset lock

`SAVE`, `BGSAVE`, `DEBUG RELOAD` and `PSYNC` need a view of the whole dataset,
while writes happen concurrently in many goroutines. To make both consistent,
the `rdb` package owns a dataset lock:

- a write command takes it exclusively from before it is executed until it has
  been propagated to the replicas
- a dump takes it exclusively as well, and serializes the stores while no write
  can interleave (each shard is copied through its own command channel, the lists
  and streams under their mutexes)

Blocking commands such as `BLPOP` or `XREAD BLOCK` are not write commands and
never hold the lock, so a blocked client cannot stall a snapshot. Commands that
run inside a `MULTI`/`EXEC` transaction are covered by the single lock `EXEC`
takes around the whole transaction.

## Limitations and Future Work

Currently Unsupported:
- AOF (append only file) persistence
- Partial resynchronization (keyforge has no replication backlog, so every
  `PSYNC` is a full resynchronization)
- Automatic saving based on thresholds (`save 900 1`)
- Handling of replicas that break or fall behind (no reconnect loop)
- Cluster mode
- Lua scripting
- Sorted sets and hash data types
- Authentication (AUTH command)
- Connection timeouts and keepalive

## Testing

Run the unit tests with:
```bash
go test ./...
```

The RDB implementation has its own tests (round trips through the real stores,
checksums, corrupt and truncated payloads, byte layout):
```bash
go test ./internal/rdb/ -v
```

The persistence and replication flow is covered by an end to end script that
starts a master and a replica and drives them with `redis-cli`:
```bash
tests/test_persistence_and_replication.sh
```

Other scripts:
```bash
bash tests/verify_xread_block.sh
bash tests/verify_range_ordering.sh
```

## Project Structure

```
keyforge/
├── app/
│   ├── main.go              # Application entry point
│   ├── master.go            # Client connection handling
│   └── replica.go           # Replication handshake and snapshot loading
├── internal/
│   ├── commands/            # Command implementations
│   ├── config/              # Runtime configuration and replication offsets
│   ├── db/                  # Database storage layer
│   ├── ds/                  # Data structures (deque)
│   ├── parser/              # RESP parser
│   ├── pubsub/              # Pub/Sub implementation
│   ├── rdb/                 # RDB snapshots (read/write, dataset dump)
│   ├── resp/                # RESP protocol types
│   ├── streams/             # Stream data structure
│   └── utils/               # Utility functions
├── tests/                   # Integration tests
├── go.mod                   # Go module definition
└── README.md                # This file
```
